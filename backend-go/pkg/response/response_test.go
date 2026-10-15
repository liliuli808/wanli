package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/wanli/backend-go/pkg/apperror"
)

func TestOk(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Ok(c, gin.H{"status": "ok"})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
}

func TestFail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, apperror.ErrResourceNotFound)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp.Success)
	assert.Equal(t, "RESOURCE_NOT_FOUND", resp.Error.Code)
}
