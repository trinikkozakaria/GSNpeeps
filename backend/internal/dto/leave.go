package dto

// CreateLeaveTypeRequest memetakan schema CreateLeaveTypeRequest pada OpenAPI.
type CreateLeaveTypeRequest struct {
	Code             string `json:"kode" validate:"required,min=1,max=50"`
	Name             string `json:"nama" validate:"required,min=1,max=150"`
	AnnualQuota      int    `json:"kuota_tahunan" validate:"gte=0"`
	Category         string `json:"kategori" validate:"required,oneof=cuti izin"`
	MaximumDays      *int   `json:"maksimal_hari" validate:"omitempty,gte=1,lte=365"`
	RequiresDocument bool   `json:"memerlukan_dokumen"`
}

// UpdateLeaveTypeRequest bersifat partial; minimal satu field wajib dikirim.
type UpdateLeaveTypeRequest struct {
	Name             *string `json:"nama" validate:"omitempty,min=1,max=150"`
	AnnualQuota      *int    `json:"kuota_tahunan" validate:"omitempty,gte=0"`
	Category         *string `json:"kategori" validate:"omitempty,oneof=cuti izin"`
	MaximumDays      *int    `json:"maksimal_hari" validate:"omitempty,gte=1,lte=365"`
	RequiresDocument *bool   `json:"memerlukan_dokumen"`
	IsActive         *bool   `json:"is_active"`
}

// DecisionRequest memetakan schema DecisionRequest. Catatan wajib saat menolak; aturan
// tersebut ditegakkan handler karena bergantung pada nilai field lain.
type DecisionRequest struct {
	Decision string `json:"keputusan" validate:"required,oneof=setujui tolak"`
	Note     string `json:"catatan" validate:"omitempty,max=1000"`
}

// DelegateRequest memetakan schema DelegateRequest.
type DelegateRequest struct {
	DelegateTo string `json:"delegate_to" validate:"required,uuid"`
	Note       string `json:"catatan" validate:"required,min=5,max=1000"`
}
