package repository

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/wanli/backend-go/internal/model"
)

func TestGormKitchenRepository_SaveProfile_Integration(t *testing.T) {
	dsn := os.Getenv("WANLI_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("未设置 WANLI_TEST_DB_DSN，跳过集成测试")
	}

	db, err := gorm.Open(postgres.Open(dsn))
	require.NoError(t, err)
	repo := NewGormKitchenRepository(db)
	ctx := context.Background()

	user := model.User{
		Email:          "kitchen-" + uuid.NewString() + "@example.com",
		HashedPassword: "x",
		Nickname:       "厨房仓储测试",
	}
	require.NoError(t, db.Create(&user).Error)
	t.Cleanup(func() {
		db.Exec("DELETE FROM users WHERE id = ?", user.ID)
	})

	initial := &model.KitchenProfile{
		UserID:           user.ID,
		OwnedIngredients: []string{"盐", "生抽"},
		OwnedTools:       []string{"炒锅"},
	}
	require.NoError(t, repo.SaveProfile(ctx, initial))

	got, err := repo.GetProfile(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"盐", "生抽"}, got.OwnedIngredients)
	assert.Equal(t, []string{"炒锅"}, got.OwnedTools)

	updated := &model.KitchenProfile{
		UserID:           user.ID,
		OwnedIngredients: []string{},
		OwnedTools:       []string{"烤箱", "空气炸锅"},
	}
	require.NoError(t, repo.SaveProfile(ctx, updated))

	got, err = repo.GetProfile(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, got.OwnedIngredients)
	assert.Empty(t, got.OwnedIngredients)
	assert.Equal(t, []string{"烤箱", "空气炸锅"}, got.OwnedTools)
}
