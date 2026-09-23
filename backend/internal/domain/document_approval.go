package domain

import (
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RequestStatus values match the PRD's §5.3/§7 enum. Unlike leave requests, this module
// does not encode "which stage" inside the status — that lives in StageIndex instead.
// Status only tracks the request's overall lifecycle.
type DocumentApprovalStatus string

const (
	DocApprovalRunning   DocumentApprovalStatus = "berjalan"
	DocApprovalApproved  DocumentApprovalStatus = "disetujui"
	DocApprovalRejected  DocumentApprovalStatus = "ditolak"
	DocApprovalCancelled DocumentApprovalStatus = "dibatalkan"
	// Blocked: the active stage's Position currently has zero active holders (PRD §13).
	// The request does not advance or fail on its own — it waits for HR to act.
	DocApprovalBlocked DocumentApprovalStatus = "bermasalah"
)

func (s DocumentApprovalStatus) Pending() bool {
	return s == DocApprovalRunning || s == DocApprovalBlocked
}

type DocumentDecision string

const (
	DocDecisionApprove DocumentDecision = "approve"
	DocDecisionReject  DocumentDecision = "reject"
)

// WorkflowTemplate + WorkflowStage — §5.1 / §7.

type WorkflowTemplate struct {
	ID        uuid.UUID
	Name      string
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type WorkflowStage struct {
	ID         uuid.UUID
	TemplateID uuid.UUID
	Order      int // urutan, 0-indexed internally; displayed 1-indexed if the frontend wants that
	StageName  string
	PositionID uuid.UUID
}

type CreateWorkflowTemplate struct {
	Name   string
	Active bool
	Stages []CreateWorkflowStage
}

type CreateWorkflowStage struct {
	Order      int
	StageName  string
	PositionID uuid.UUID
}

type UpdateWorkflowTemplate struct {
	Name   *string
	Active *bool
	// Stages, when non-nil, fully replaces the existing stage list (add/remove/reorder).
	// A partial per-stage PATCH is deliberately not supported — the PRD's form page (§9.7)
	// always submits the complete stage array.
	Stages *[]CreateWorkflowStage
}

// ApprovalRequest — §7 document_approval_requests.

type ApprovalRequest struct {
	ID                    uuid.UUID
	TemplateID            uuid.UUID
	DocumentURL           string
	DocumentURLNormalized string
	Title                 string
	RequesterUserID       uuid.UUID
	StageIndex            int
	Status                DocumentApprovalStatus
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type CreateApprovalRequest struct {
	TemplateID      uuid.UUID
	DocumentURL     string
	Title           string
	RequesterUserID uuid.UUID
}

// ApprovalRequestLock is what LockRequestForDecision returns — enough state to authorize
// and apply a decision without a second round trip.
type ApprovalRequestLock struct {
	RequestID       uuid.UUID
	TemplateID      uuid.UUID
	RequesterUserID uuid.UUID
	StageIndex      int
	Status          DocumentApprovalStatus
	TotalStages     int
}

type ApprovalDecisionRecord struct {
	StageOrder     int
	ApproverUserID uuid.UUID
	ApproverName   string
	Decision       DocumentDecision
	Note           *string
	DecidedAt      time.Time
}

type ApprovalRequestDetail struct {
	ApprovalRequest
	TemplateName string
	Stages       []StageProgress
	History      []ApprovalDecisionRecord
}

// StageProgress is what the detail page (§9.4) renders: each stage's position, its live
// resolved holders, and whether it's done/active/pending relative to StageIndex.
type StageProgress struct {
	Order        int
	StageName    string
	PositionID   uuid.UUID
	PositionName string
	Holders      []StageApprover // live-resolved, can be empty
	Done         bool
	Active       bool
}

type StageApprover struct {
	UserID       uuid.UUID
	EmployeeName string
}

type ApprovalRequestScope struct {
	RequesterUserID *uuid.UUID
	// ApproverUserID, when set, restricts the list to requests whose *current* active
	// stage resolves (live) to include this user as a holder — computed in the service
	// layer, not the repository, since it depends on the live employees/positions join.
	ApproverUserID *uuid.UUID
}

type ApprovalRequestFilter struct {
	Scope      ApprovalRequestScope
	Status     *DocumentApprovalStatus
	TemplateID *uuid.UUID
	Page, Limit int
}

type ApprovalRequestPage struct {
	Items []ApprovalRequest
	Total int
	Page, Limit int
}

// NormalizeDocumentURL implements §5.2's normalization rule: trim whitespace and
// lowercase the host, so that differently-written forms of the same URL collide for
// idempotency purposes. Deliberately does not lowercase the full URL (path/query are
// often case-sensitive on real servers, e.g. Nextcloud share tokens).
func NormalizeDocumentURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrInvalidRequest
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", ErrInvalidRequest
	}
	parsed.Host = strings.ToLower(parsed.Host)
	// Strip a trailing slash on the bare path so "example.com/doc" and "example.com/doc/"
	// normalize to the same key.
	if parsed.Path != "/" {
		parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	}
	return parsed.String(), nil
}