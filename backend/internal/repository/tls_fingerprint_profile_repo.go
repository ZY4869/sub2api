package repository

import (
	"context"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/tlsfingerprintprofile"
	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type tlsFingerprintProfileRepository struct {
	client *ent.Client
}

func NewTLSFingerprintProfileRepository(client *ent.Client) service.TLSFingerprintProfileRepository {
	return &tlsFingerprintProfileRepository{client: client}
}

func (r *tlsFingerprintProfileRepository) List(ctx context.Context) ([]*model.TLSFingerprintProfile, error) {
	profiles, err := r.client.TLSFingerprintProfile.Query().
		Order(ent.Asc(tlsfingerprintprofile.FieldName)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*model.TLSFingerprintProfile, 0, len(profiles))
	for _, profile := range profiles {
		result = append(result, r.toModel(profile))
	}
	return result, nil
}

func (r *tlsFingerprintProfileRepository) GetByID(ctx context.Context, id int64) (*model.TLSFingerprintProfile, error) {
	profile, err := r.client.TLSFingerprintProfile.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(profile), nil
}

func (r *tlsFingerprintProfileRepository) Create(ctx context.Context, profile *model.TLSFingerprintProfile) (*model.TLSFingerprintProfile, error) {
	builder := r.client.TLSFingerprintProfile.Create().
		SetName(profile.Name).
		SetEnableGrease(profile.EnableGREASE)

	if profile.Description != nil {
		builder.SetDescription(*profile.Description)
	}
	if len(profile.CipherSuites) > 0 {
		builder.SetCipherSuites(profile.CipherSuites)
	}
	if len(profile.Curves) > 0 {
		builder.SetCurves(profile.Curves)
	}
	if len(profile.PointFormats) > 0 {
		builder.SetPointFormats(profile.PointFormats)
	}
	if len(profile.SignatureAlgorithms) > 0 {
		builder.SetSignatureAlgorithms(profile.SignatureAlgorithms)
	}
	if len(profile.ALPNProtocols) > 0 {
		builder.SetAlpnProtocols(profile.ALPNProtocols)
	}
	if len(profile.SupportedVersions) > 0 {
		builder.SetSupportedVersions(profile.SupportedVersions)
	}
	if len(profile.KeyShareGroups) > 0 {
		builder.SetKeyShareGroups(profile.KeyShareGroups)
	}
	if len(profile.PSKModes) > 0 {
		builder.SetPskModes(profile.PSKModes)
	}
	if len(profile.Extensions) > 0 {
		builder.SetExtensions(profile.Extensions)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	return r.toModel(created), nil
}

func (r *tlsFingerprintProfileRepository) Update(ctx context.Context, profile *model.TLSFingerprintProfile) (*model.TLSFingerprintProfile, error) {
	builder := r.client.TLSFingerprintProfile.UpdateOneID(profile.ID).
		SetName(profile.Name).
		SetEnableGrease(profile.EnableGREASE)

	if profile.Description != nil {
		builder.SetDescription(*profile.Description)
	} else {
		builder.ClearDescription()
	}
	if len(profile.CipherSuites) > 0 {
		builder.SetCipherSuites(profile.CipherSuites)
	} else {
		builder.ClearCipherSuites()
	}
	if len(profile.Curves) > 0 {
		builder.SetCurves(profile.Curves)
	} else {
		builder.ClearCurves()
	}
	if len(profile.PointFormats) > 0 {
		builder.SetPointFormats(profile.PointFormats)
	} else {
		builder.ClearPointFormats()
	}
	if len(profile.SignatureAlgorithms) > 0 {
		builder.SetSignatureAlgorithms(profile.SignatureAlgorithms)
	} else {
		builder.ClearSignatureAlgorithms()
	}
	if len(profile.ALPNProtocols) > 0 {
		builder.SetAlpnProtocols(profile.ALPNProtocols)
	} else {
		builder.ClearAlpnProtocols()
	}
	if len(profile.SupportedVersions) > 0 {
		builder.SetSupportedVersions(profile.SupportedVersions)
	} else {
		builder.ClearSupportedVersions()
	}
	if len(profile.KeyShareGroups) > 0 {
		builder.SetKeyShareGroups(profile.KeyShareGroups)
	} else {
		builder.ClearKeyShareGroups()
	}
	if len(profile.PSKModes) > 0 {
		builder.SetPskModes(profile.PSKModes)
	} else {
		builder.ClearPskModes()
	}
	if len(profile.Extensions) > 0 {
		builder.SetExtensions(profile.Extensions)
	} else {
		builder.ClearExtensions()
	}

	updated, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	return r.toModel(updated), nil
}

func (r *tlsFingerprintProfileRepository) Delete(ctx context.Context, id int64) error {
	return r.client.TLSFingerprintProfile.DeleteOneID(id).Exec(ctx)
}

func (r *tlsFingerprintProfileRepository) toModel(entity *ent.TLSFingerprintProfile) *model.TLSFingerprintProfile {
	if entity == nil {
		return nil
	}
	profile := &model.TLSFingerprintProfile{
		ID:                  entity.ID,
		Name:                entity.Name,
		Description:         entity.Description,
		EnableGREASE:        entity.EnableGrease,
		CipherSuites:        entity.CipherSuites,
		Curves:              entity.Curves,
		PointFormats:        entity.PointFormats,
		SignatureAlgorithms: entity.SignatureAlgorithms,
		ALPNProtocols:       entity.AlpnProtocols,
		SupportedVersions:   entity.SupportedVersions,
		KeyShareGroups:      entity.KeyShareGroups,
		PSKModes:            entity.PskModes,
		Extensions:          entity.Extensions,
		CreatedAt:           entity.CreatedAt,
		UpdatedAt:           entity.UpdatedAt,
	}
	if profile.CipherSuites == nil {
		profile.CipherSuites = []uint16{}
	}
	if profile.Curves == nil {
		profile.Curves = []uint16{}
	}
	if profile.PointFormats == nil {
		profile.PointFormats = []uint16{}
	}
	if profile.SignatureAlgorithms == nil {
		profile.SignatureAlgorithms = []uint16{}
	}
	if profile.ALPNProtocols == nil {
		profile.ALPNProtocols = []string{}
	}
	if profile.SupportedVersions == nil {
		profile.SupportedVersions = []uint16{}
	}
	if profile.KeyShareGroups == nil {
		profile.KeyShareGroups = []uint16{}
	}
	if profile.PSKModes == nil {
		profile.PSKModes = []uint16{}
	}
	if profile.Extensions == nil {
		profile.Extensions = []uint16{}
	}
	return profile
}
