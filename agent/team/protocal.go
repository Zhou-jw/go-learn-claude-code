package team

import (
	"sync"

	"github.com/google/uuid"
)

// 协议类型
const (
	ProtocolShutdown     = "shutdown"
	ProtocolPlanApproval = "plan_approval"
)

// 响应结果
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

// ==============================================
// Request：一次协议请求（核心）
// ==============================================

type Request struct {
	ID        string
	Type      string
	From      string
	To        string
	Status    string
}

// Response：协议响应结构
type Response struct {
	Approve  bool
}

// ==============================================
// Protocol：协议管理器（不依赖全局变量！）
// ==============================================

type Protocol struct {
	mu sync.Mutex
	requests map[string]*Request // 这里的 map 只存 index，不存状态
}

func NewProtocol() *Protocol {
	return &Protocol{
		requests: make(map[string]*Request),
	}
}

// ==============================================
// 1. 创建请求（Lead/Teammate 调用）
// ==============================================

func (p *Protocol) NewRequest(reqType, from, to string) *Request {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	reqID := uuid.NewString()[:8]

	req := &Request{
		ID:           reqID,
		Type:         reqType,
		From:         from,
		To:           to,
		Status:       StatusPending,
	}

	// 注册到 index map
	p.requests[reqID] = req
	return req
}

func (p *Protocol) GetRequest(reqID string) (*Request, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	req, ok := p.requests[reqID]
	return req, ok
}

func (p *Protocol) Update(req_id string, approve bool) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	req, ok := p.GetRequest(req_id)
	if !ok {
		return false
	}

	if approve {
		req.Status = StatusApproved
	} else {
		req.Status = StatusRejected
	}
	return true
}