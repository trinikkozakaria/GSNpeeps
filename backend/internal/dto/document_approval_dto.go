package dto

// CreateWorkflowTemplateRequest — POST /master/alur-persetujuan-dokumen (PRD §9.7).
type CreateWorkflowTemplateRequest struct {
	Name   string                        `json:"nama" validate:"required,min=3,max=150"`
	Active bool                          `json:"aktif"`
	Stages []CreateWorkflowStageRequest  `json:"tahap" validate:"required,min=1,dive"`
}

type CreateWorkflowStageRequest struct {
	Order      int    `json:"urutan"`
	StageName  string `json:"nama_tahap" validate:"required,min=3,max=150"`
	PositionID string `json:"position_id" validate:"required,uuid"`
}

// UpdateWorkflowTemplateRequest — PUT /master/alur-persetujuan-dokumen/{id}. Stages, when
// present, replaces the full stage list (no partial per-stage patch — PRD §9.7 always
// submits the complete array).
type UpdateWorkflowTemplateRequest struct {
	Name   *string                       `json:"nama" validate:"omitempty,min=3,max=150"`
	Active *bool                         `json:"aktif"`
	Stages *[]CreateWorkflowStageRequest `json:"tahap" validate:"omitempty,min=1,dive"`
}

// CreateApprovalRequestRequest — POST /persetujuan-dokumen (PRD §9.1). Plain JSON, not
// multipart — the document itself is never uploaded, only its URL is stored (PRD §2
// Non-Goals: no document content storage).
type CreateApprovalRequestRequest struct {
	DocumentURL string `json:"document_url" validate:"required,url"`
	Title       string `json:"judul" validate:"required,min=3,max=255"`
	TemplateID  string `json:"template_id" validate:"required,uuid"`
}

// DocumentDecisionRequest — PUT /persetujuan-dokumen/{id}/decision (PRD §5.3/§9.4). Note
// the enum values are literally "approve"/"reject" per the PRD — unlike leave's
// "setujui"/"tolak" — this is not a naming inconsistency, it's what the spec asked for.
type DocumentDecisionRequest struct {
	Keputusan string `json:"keputusan" validate:"required,oneof=approve reject"`
	Catatan   string `json:"catatan"`
}