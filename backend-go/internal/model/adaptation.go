package model

import "fmt"

// AdaptationResult 是 AI 厨房适配的结构化输出。
type AdaptationResult struct {
	Safe             bool               `json:"safe"`
	BlockReason      *string            `json:"block_reason"`
	Title            string             `json:"title"`
	Servings         *string            `json:"servings"`
	EstimatedMinutes *int               `json:"estimated_minutes"`
	Ingredients      []ParsedIngredient `json:"ingredients"`
	Steps            []ParsedStep       `json:"steps"`
	Instructions     *string            `json:"instructions"`
	AdaptationNotes  AdaptationNotes    `json:"adaptation_notes"`
}

type AdaptationNotes struct {
	MissingIngredients []string                 `json:"missing_ingredients"`
	MissingTools       []string                 `json:"missing_tools"`
	Substitutions      []AdaptationSubstitution `json:"substitutions"`
	FlavorChanges      []string                 `json:"flavor_changes"`
}

type AdaptationSubstitution struct {
	Original    string `json:"original"`
	Replacement string `json:"replacement"`
	Reason      string `json:"reason"`
}

func (r *AdaptationResult) ParsedRecipe() *ParsedRecipe {
	return &ParsedRecipe{
		Title:            r.Title,
		Servings:         r.Servings,
		EstimatedMinutes: r.EstimatedMinutes,
		Ingredients:      r.Ingredients,
		Steps:            r.Steps,
		Instructions:     r.Instructions,
	}
}

func (r *AdaptationResult) Validate() error {
	if !r.Safe {
		if r.BlockReason == nil || *r.BlockReason == "" {
			return fmt.Errorf("safe=false 时 block_reason 必填")
		}
		return nil
	}
	return r.ParsedRecipe().Validate()
}
