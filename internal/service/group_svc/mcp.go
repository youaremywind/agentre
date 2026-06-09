package group_svc

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type memberRef struct{ groupID, memberID int64 }

// groupMCP 是 group_send tool 的 MCP-over-HTTP server(挂在 gateway /mcp/group/)。
//
// 身份: 无状态签名 token —— `b64url(group:member).b64url(HMAC(secret, group:member))`,
// 投递时塞进 mcp-config 的 Authorization header。token 只在 CLI spawn 时随 --mcp-config
// 注入、复用轮不会重发,因此必须与子进程同寿命:确定性(同成员每次同值)+ 跨重启/恢复稳定。
// 旧实现用内存 map 存随机 token + 停止/归档/离群时 delete,但被复用的常驻子进程仍持旧 token,
// 一旦 map 里没了(停止吊销 / 进程重启)group_send 就拿 401,被 CLI 误报"需要重新授权"。
// 现改为:lookup 只验签(无状态),发言权由 authorized 按 DB 成员资格实时判定(见 memberCanPost)。
type groupMCP struct {
	secret []byte // per-process HMAC 签名密钥(群聊为本机回投,进程内即可;重启随子进程池一并重建)
	ingest func(ctx context.Context, memberID int64, body string, mentions []string) error
	invite func(ctx context.Context, memberID int64, names []string, ids []int64, reason string) ([]InviteResult, error)
	authz  func(ctx context.Context, groupID, memberID int64) bool // 发言权判定;nil=放行(测试默认)
}

func newGroupMCP(ingest func(context.Context, int64, string, []string) error) *groupMCP {
	return &groupMCP{secret: randSecret(), ingest: ingest}
}

// MintToken 为某成员签一个绑定 (group, member) 的无状态签名 token。同一 (group, member)
// 每次返回相同值(确定性),保证被复用/恢复的子进程持有的 token 始终验签通过。
func (h *groupMCP) MintToken(groupID, memberID int64) string {
	payload := strconv.FormatInt(groupID, 10) + ":" + strconv.FormatInt(memberID, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + h.sign(payload)
}

func (h *groupMCP) sign(payload string) string {
	mac := hmac.New(sha256.New, h.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// lookup 验签并解出 token 绑定的 (group, member)。仅做密码学校验(无状态),不查成员资格 ——
// 是否仍有发言权(离群 / 归档)由 authorized 按 DB 现状判定。验签失败 / 格式非法 → !ok。
func (h *groupMCP) lookup(tok string) (memberRef, bool) {
	payloadB64, sig, ok := strings.Cut(tok, ".")
	if !ok {
		return memberRef{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil || !hmac.Equal([]byte(h.sign(string(payload))), []byte(sig)) {
		return memberRef{}, false
	}
	gStr, mStr, ok := strings.Cut(string(payload), ":")
	if !ok {
		return memberRef{}, false
	}
	groupID, err1 := strconv.ParseInt(gStr, 10, 64)
	memberID, err2 := strconv.ParseInt(mStr, 10, 64)
	if err1 != nil || err2 != nil {
		return memberRef{}, false
	}
	return memberRef{groupID, memberID}, true
}

// authorized 报告 (group, member) 当前是否仍可发言。authz 未装配(测试默认)→ 放行。
func (h *groupMCP) authorized(ctx context.Context, groupID, memberID int64) bool {
	return h.authz == nil || h.authz(ctx, groupID, memberID)
}

func (h *groupMCP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet { // claude 开 server→client SSE; 我们不推送 → 405(claude 容忍)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var rpc struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params struct {
			ProtocolVersion string `json:"protocolVersion"`
			Name            string `json:"name"`
			Arguments       struct {
				Body       string   `json:"body"`
				Mentions   []string `json:"mentions"`
				AgentNames []string `json:"agentNames"`
				AgentIDs   []int64  `json:"agentIds"`
				Reason     string   `json:"reason"`
			} `json:"arguments"`
		} `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&rpc); err != nil {
		writeRPCError(w, nil, -32700, "parse error")
		return
	}
	switch rpc.Method {
	case "initialize":
		pv := rpc.Params.ProtocolVersion
		if pv == "" {
			pv = "2025-06-18"
		}
		writeRPCResult(w, rpc.ID, map[string]any{
			"protocolVersion": pv,
			"serverInfo":      map[string]any{"name": "agentre-group", "version": "1"},
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
		})
	case "notifications/initialized":
		w.WriteHeader(http.StatusAccepted)
	case "tools/list":
		writeRPCResult(w, rpc.ID, map[string]any{"tools": []any{groupSendToolSchema(), groupInviteToolSchema()}})
	case "tools/call":
		ref, ok := h.lookup(bearer(r))
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !h.authorized(r.Context(), ref.groupID, ref.memberID) { // 离群 / 群归档即失权(按 DB 现状)
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		switch rpc.Params.Name {
		case "group_send":
			if h.ingest == nil { // 防御: 生产装配后应始终非 nil
				writeRPCError(w, rpc.ID, -32000, "ingest not wired")
				return
			}
			if err := h.ingest(r.Context(), ref.memberID, rpc.Params.Arguments.Body, rpc.Params.Arguments.Mentions); err != nil {
				writeRPCError(w, rpc.ID, -32000, err.Error())
				return
			}
			writeRPCResult(w, rpc.ID, map[string]any{"content": []any{map[string]any{"type": "text", "text": "sent"}}})
		case "group_invite":
			if h.invite == nil {
				writeRPCError(w, rpc.ID, -32000, "invite not wired")
				return
			}
			results, err := h.invite(r.Context(), ref.memberID, rpc.Params.Arguments.AgentNames, rpc.Params.Arguments.AgentIDs, rpc.Params.Arguments.Reason)
			if err != nil {
				writeRPCError(w, rpc.ID, -32000, err.Error())
				return
			}
			names := make([]string, 0, len(results))
			for _, x := range results {
				names = append(names, x.Name)
			}
			writeRPCResult(w, rpc.ID, map[string]any{"content": []any{map[string]any{"type": "text", "text": "invited: " + strings.Join(names, ", ")}}})
		default:
			writeRPCError(w, rpc.ID, -32601, "unknown tool")
		}
	default:
		writeRPCError(w, rpc.ID, -32601, "method not found")
	}
}

func bearer(r *http.Request) string {
	return strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
}

func groupSendToolSchema() map[string]any {
	return map[string]any{
		"name":        "group_send",
		"description": "向群聊发送一条消息。mentions 填收件成员的显示名(@用户 = 回复人类)。一个回合可多次调用。",
		"inputSchema": map[string]any{
			"type":     "object",
			"required": []string{"body"},
			"properties": map[string]any{
				"body":     map[string]any{"type": "string", "description": "消息正文"},
				"mentions": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "收件成员显示名"},
			},
		},
	}
}

func groupInviteToolSchema() map[string]any {
	return map[string]any{
		"name":        "group_invite",
		"description": "把本部门的 Agent 拉进当前群聊。只有主持人可调用。agentNames 或 agentIds 二选一。",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"agentNames": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "被邀请成员显示名"},
				"agentIds":   map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "description": "被邀请成员 agent id"},
				"reason":     map[string]any{"type": "string", "description": "邀请理由(可选)"},
			},
		},
	}
}

func writeRPCResult(w http.ResponseWriter, id json.RawMessage, result any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func writeRPCError(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": msg}})
}

// randSecret 生成本进程的 HMAC 签名密钥(32 字节)。crypto/rand 失败是不可恢复的灾难;
// 签名密钥绝不能退化为可预测值, 必须 fail loud。
func randSecret() []byte {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("group_svc: crypto/rand failed: " + err.Error())
	}
	return b
}
