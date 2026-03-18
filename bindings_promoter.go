package main

import (
	"fmt"

	"lurus-switch/internal/promoter"
)

// PromoterGetInfo retrieves promoter information (aff_code, referral stats, share link).
func (a *App) PromoterGetInfo() (*promoter.PromoterInfo, error) {
	if a.promoterSvc == nil {
		return nil, fmt.Errorf("promoter service not initialized")
	}
	return a.promoterSvc.GetInfo(a.ctx)
}

// PromoterGetShareLink returns the share URL for the current user's aff_code.
func (a *App) PromoterGetShareLink() (string, error) {
	if a.promoterSvc == nil {
		return "", fmt.Errorf("promoter service not initialized")
	}
	info, err := a.promoterSvc.GetInfo(a.ctx)
	if err != nil {
		return "", err
	}
	return info.ShareLink, nil
}
