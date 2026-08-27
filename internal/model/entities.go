// Package model 定义环境 DNA 采样空白传播诊断服务的领域实体、状态枚举与错误。
package model

import "time"

// ChainStatus 实验链状态机：接收中 → 待追溯 → 需复核 → 已发布 → 封存。
type ChainStatus string

const (
	ChainReceiving    ChainStatus = "receiving"
	ChainTracePending ChainStatus = "trace_pending"
	ChainNeedsReview  ChainStatus = "needs_review"
	ChainPublished    ChainStatus = "published"
	ChainSealed       ChainStatus = "sealed"
)

// FeatureStatus 序列特征记录状态。
type FeatureStatus string

const (
	FeatureRaw           FeatureStatus = "raw"
	FeatureFieldSupported FeatureStatus = "field_supported"
	FeatureBlankRelated  FeatureStatus = "blank_related"
	FeatureLowQuality    FeatureStatus = "low_quality"
	FeatureExcluded      FeatureStatus = "excluded"
)

// PathStatus 污染路径状态机：候选 → 批次相关 → 现场可信 → 确认 / 否决。
type PathStatus string

const (
	PathCandidate     PathStatus = "candidate"
	PathBatchRelated  PathStatus = "batch_related"
	PathFieldCredible PathStatus = "field_credible"
	PathConfirmed     PathStatus = "confirmed"
	PathRejected      PathStatus = "rejected"
)

// SnapshotStatus 可信度快照状态机：草稿 → 发布 → 替代。
type SnapshotStatus string

const (
	SnapDraft      SnapshotStatus = "draft"
	SnapPublished  SnapshotStatus = "published"
	SnapSuperseded SnapshotStatus = "superseded"
)

// BlankKind 空白对照种类。
type BlankKind string

const (
	BlankExtraction BlankKind = "extraction_blank"
	BlankPCR        BlankKind = "pcr_blank"
	BlankField      BlankKind = "field_blank"
)

// SourceType 特征来源类型：现场样本 or 空白对照。
type SourceType string

const (
	SourceSample SourceType = "sample"
	SourceBlank  SourceType = "blank"
)

// ChainStepRole 实验链步骤角色。
type ChainStepRole string

const (
	RoleFieldSample        ChainStepRole = "field_sample"
	RoleExtractionBlank    ChainStepRole = "extraction_blank"
	RoleAmplificationBatch ChainStepRole = "amplification_batch"
)

// Sample 现场滤膜样本。
type Sample struct {
	ID         string
	Code       string
	Site       string
	Matrix     string // 介质：water/sediment/soil
	CollectedAt time.Time
	Notes      string
	CreatedAt  time.Time
}

// Blank 空白对照。
type Blank struct {
	ID       string
	Code     string
	Kind     BlankKind
	Notes    string
	CreatedAt time.Time
}

// Batch 扩增批次。
type Batch struct {
	ID        string
	Code      string
	Plate     string
	RunAt     time.Time
	Isolated  bool // 隔离后不再参与传播
	CreatedAt time.Time
}

// Feature 序列特征观察（某分类单元在某来源、某批次下的读段）。
type Feature struct {
	ID         string
	Taxon      string // 物种/分类单元标识
	Marker     string // 遗传标记：12S/COI/18S
	ReadCount  int
	SourceType SourceType
	SourceID   string // sample_id 或 blank_id
	BatchID    string
	Quality    float64 // 0..1
	Status     FeatureStatus
	CreatedAt  time.Time
}

// Chain 实验链：将样本、空白、批次按实验步骤关联。
type Chain struct {
	ID        string
	Name      string
	Status    ChainStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ChainStep 实验链步骤（有向依赖：样本依赖批次，空白依赖批次）。
type ChainStep struct {
	ID         string
	ChainID    string
	EntityType string // sample|blank|batch
	EntityID   string
	Role       ChainStepRole
	OrderIdx   int
}

// ContamPath 污染路径候选：某分类单元经某空白、某批次向样本传播的证据。
type ContamPath struct {
	ID         string
	ChainID    string
	Taxon      string
	ViaBlankID string
	ViaBatchID string
	Status     PathStatus
	Score      float64 // 污染可疑度 0..1
	Evidence   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Snapshot 可信度快照：每个分类单元在现场信号中的可信度。
type Snapshot struct {
	ID          string
	ChainID     string
	Status      SnapshotStatus
	Threshold   float64 // 可信度阈值，低于则判为可疑
	Payload     string  // JSON：分类单元 -> 可信度与证据
	CreatedAt   time.Time
	PublishedAt time.Time
}

// TaxonCredibility 单个分类单元的可信度判定结果。
type TaxonCredibility struct {
	Taxon       string   `json:"taxon"`
	Marker      string   `json:"marker"`
	Credibility float64  `json:"credibility"`
	Suspect     bool     `json:"suspect"`
	PathCount   int      `json:"path_count"`
	Evidence    string   `json:"evidence"`
}

// SnapshotPayload 快照载荷结构。
type SnapshotPayload struct {
	ChainID  string             `json:"chain_id"`
	Taxa     []TaxonCredibility `json:"taxa"`
	GeneratedAt string          `json:"generated_at"`
}
