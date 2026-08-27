// Package sample 负责实验输入数据的录入与校验：样本、空白、批次与序列特征。
package sample

import (
	"regexp"
	"time"

	"task282-ednadiag/internal/model"
	"task282-ednadiag/internal/store"
)

var asciiRe = regexp.MustCompile(`^[\x20-\x7E]+$`)

// Service 样本录入服务。
type Service struct {
	store *store.Store
}

// NewService 构造样本服务。
func NewService(s *store.Store) *Service { return &Service{store: s} }

func validCode(s string) bool { return s != "" && asciiRe.MatchString(s) && len(s) <= 128 }

// CreateSample 录入现场滤膜样本。
func (svc *Service) CreateSample(smp *model.Sample) error {
	if smp.ID == "" || !validCode(smp.Code) || smp.Site == "" || smp.Matrix == "" {
		return model.ErrInvalidInput
	}
	if smp.CollectedAt.IsZero() {
		smp.CollectedAt = time.Now()
	}
	return svc.store.CreateSample(smp)
}

// CreateBlank 录入空白对照。
func (svc *Service) CreateBlank(b *model.Blank) error {
	if b.ID == "" || !validCode(b.Code) {
		return model.ErrInvalidInput
	}
	switch b.Kind {
	case model.BlankExtraction, model.BlankPCR, model.BlankField:
	default:
		return model.ErrInvalidInput
	}
	return svc.store.CreateBlank(b)
}

// CreateBatch 录入扩增批次。
func (svc *Service) CreateBatch(b *model.Batch) error {
	if b.ID == "" || !validCode(b.Code) || b.Plate == "" {
		return model.ErrInvalidInput
	}
	if b.RunAt.IsZero() {
		b.RunAt = time.Now()
	}
	return svc.store.CreateBatch(b)
}

// CreateFeature 录入序列特征：校验编码合法性、批次存在性、来源存在性。
func (svc *Service) CreateFeature(f *model.Feature) error {
	if f.ID == "" {
		return model.ErrInvalidInput
	}
	if f.Taxon == "" || f.Marker == "" {
		return model.ErrFeatureCodeInvalid
	}
	if !asciiRe.MatchString(f.Taxon) || !asciiRe.MatchString(f.Marker) {
		return model.ErrFeatureCodeInvalid
	}
	if f.ReadCount < 0 || f.Quality < 0 || f.Quality > 1 {
		return model.ErrInvalidInput
	}
	switch f.SourceType {
	case model.SourceSample:
		ok, err := svc.store.SampleExists(f.SourceID)
		if err != nil {
			return err
		}
		if !ok {
			return model.ErrInvalidInput
		}
	case model.SourceBlank:
		ok, err := svc.store.BlankExists(f.SourceID)
		if err != nil {
			return err
		}
		if !ok {
			return model.ErrInvalidInput
		}
	default:
		return model.ErrInvalidInput
	}
	ok, err := svc.store.BatchExists(f.BatchID)
	if err != nil {
		return err
	}
	if !ok {
		return model.ErrBatchMissing
	}
	return svc.store.CreateFeature(f)
}
