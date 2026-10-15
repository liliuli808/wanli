package model

import "fmt"

// 食材分类枚举，与 docs/ai-parsing.md 及 recipe_ingredients.category 保持一致。
const (
	IngredientCategoryMain      = "MAIN"      // 主料
	IngredientCategoryAuxiliary = "AUXILIARY" // 辅料
	IngredientCategorySeasoning = "SEASONING" // 调味品
)

// ParsedRecipe 是 AI 从视频文本中解析出的结构化菜谱（对应 docs/ai-parsing.md §2.2）。
// 可空字段使用指针，忠实保留 AI 输出的 null（禁止强行估算）。
type ParsedRecipe struct {
	Title            string             `json:"title"`
	Servings         *string            `json:"servings"`
	EstimatedMinutes *int               `json:"estimated_minutes"`
	Ingredients      []ParsedIngredient `json:"ingredients"`
	Steps            []ParsedStep       `json:"steps"`
	Instructions     *string            `json:"instructions"`
}

// ParsedIngredient 是一个解析出的食材/调料项。
type ParsedIngredient struct {
	Name          string  `json:"name"`
	Amount        *string `json:"amount"` // 不确定为 null；“少许/适量”原样保留
	Category      string  `json:"category"`
	IsExplicit    bool    `json:"is_explicit"` // false = AI 基于常识补充
	SubstituteFor *string `json:"substitute_for,omitempty"`
}

// ParsedStep 是一个解析出的烹饪步骤。
type ParsedStep struct {
	StepOrder       int     `json:"step_order"`
	Description     string  `json:"description"`
	VideoTimestamp  *int    `json:"video_timestamp"`  // 秒；无字幕来源时为 null
	DurationSeconds *int    `json:"duration_seconds"` // 秒
	Heat            *string `json:"heat"`             // 如“大火”“中小火”
}

// Validate 校验 AI 输出是否满足 schema 的关键约束（对应 ai-parsing.md §2.4 的失败判定）。
func (r *ParsedRecipe) Validate() error {
	if r.Title == "" {
		return fmt.Errorf("title 为空")
	}
	if len(r.Ingredients) == 0 {
		return fmt.Errorf("ingredients 为空")
	}
	for i, ing := range r.Ingredients {
		if ing.Name == "" {
			return fmt.Errorf("ingredients[%d].name 为空", i)
		}
		switch ing.Category {
		case IngredientCategoryMain, IngredientCategoryAuxiliary, IngredientCategorySeasoning:
		default:
			return fmt.Errorf("ingredients[%d].category 非法: %q", i, ing.Category)
		}
	}
	if len(r.Steps) == 0 {
		return fmt.Errorf("steps 为空")
	}
	for i, step := range r.Steps {
		if step.StepOrder < 1 {
			return fmt.Errorf("steps[%d].step_order 非法: %d", i, step.StepOrder)
		}
		if step.Description == "" {
			return fmt.Errorf("steps[%d].description 为空", i)
		}
	}
	return nil
}
