package main

import (
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"lurus-switch/internal/promptlib"
)

// ============================
// Prompt Library Methods (Phase F)
// ============================

// ListPrompts returns prompts optionally filtered by category
func (a *App) ListPrompts(category string) ([]promptlib.Prompt, error) {
	if a.promptStr == nil {
		return promptlib.GetBuiltinPrompts(), nil
	}
	return a.promptStr.ListPrompts(category)
}

// SavePrompt persists a prompt
func (a *App) SavePrompt(p promptlib.Prompt) error {
	if a.promptStr == nil {
		return fmt.Errorf("prompt store not initialized")
	}
	return a.promptStr.SavePrompt(p)
}

// DeletePrompt removes a prompt by ID
func (a *App) DeletePrompt(id string) error {
	if a.promptStr == nil {
		return fmt.Errorf("prompt store not initialized")
	}
	return a.promptStr.DeletePrompt(id)
}

// GetBuiltinPrompts returns the bundled prompt library
func (a *App) GetBuiltinPrompts() []promptlib.Prompt {
	return promptlib.GetBuiltinPrompts()
}

// ExportPrompts serializes all prompts and triggers a save dialog
func (a *App) ExportPrompts() (string, error) {
	if a.promptStr == nil {
		return "", fmt.Errorf("prompt store not initialized")
	}
	data, err := a.promptStr.ExportAll()
	if err != nil {
		return "", err
	}
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export Prompts",
		DefaultFilename: "prompts-export.json",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON", Pattern: "*.json"},
		},
	})
	if err != nil {
		return "", err
	}
	if savePath == "" {
		return "", fmt.Errorf("no save location selected")
	}
	if err := os.WriteFile(savePath, []byte(data), 0644); err != nil {
		return "", fmt.Errorf("failed to write export file: %w", err)
	}
	return savePath, nil
}

// ImportPrompts opens a file picker and imports prompts from JSON
func (a *App) ImportPrompts() (int, error) {
	if a.promptStr == nil {
		return 0, fmt.Errorf("prompt store not initialized")
	}
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Import Prompts",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON", Pattern: "*.json"},
		},
	})
	if err != nil {
		return 0, err
	}
	if filePath == "" {
		return 0, fmt.Errorf("no file selected")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read import file: %w", err)
	}
	return a.promptStr.ImportFromJSON(string(data))
}
