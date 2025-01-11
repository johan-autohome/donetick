package chore

import (
	"net/http"
	"strconv"
	"time"

	"donetick.com/core/config"
	chRepo "donetick.com/core/internal/chore/repo"
	"donetick.com/core/internal/utils"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"

	limiter "github.com/ulule/limiter/v3"

	uRepo "donetick.com/core/internal/user/repo"
)

type API struct {
	choreRepo *chRepo.ChoreRepository
	userRepo  *uRepo.UserRepository
}

func NewAPI(cr *chRepo.ChoreRepository, userRepo *uRepo.UserRepository) *API {
	return &API{
		choreRepo: cr,
		userRepo:  userRepo,
	}
}

func (h *API) GetAllChores(c *gin.Context) {

	apiToken := c.GetHeader("secretkey")
	if apiToken == "" {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	user, err := h.userRepo.GetUserByToken(c, apiToken)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	chores, err := h.choreRepo.GetChores(c, user.CircleID, user.ID, false)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, chores)
}

func (h *API) PostCompleteChore(c *gin.Context) {
	apiToken := c.GetHeader("secretkey")
	if apiToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	user, err := h.userRepo.GetUserByToken(c, apiToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")
	choreId, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad id"})
		return
	}

	chore, err := h.choreRepo.GetChore(c, choreId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chore NotFound"})
		return
	}

	choreHistory, err := h.choreRepo.GetChoreHistory(c, choreId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting history"})
		return
	}

	nextAssignedTo, err := checkNextAssignee(chore, choreHistory, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking next assignee"})
		return
	}

	completedAt := time.Now()
	err = h.choreRepo.CompleteChore(c, chore, nil, user.ID, chore.NextDueDate, &completedAt, nextAssignedTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error completing chore"})
		return
	}

	chore, err = h.choreRepo.GetChore(c, choreId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting chore"})
		return
	}
	c.JSON(200, chore)
}

func APIs(cfg *config.Config, api *API, r *gin.Engine, auth *jwt.GinJWTMiddleware, limiter *limiter.Limiter) {

	thingsAPI := r.Group("eapi/v1/chore")

	thingsAPI.Use(utils.TimeoutMiddleware(cfg.Server.WriteTimeout), utils.RateLimitMiddleware(limiter))
	{
		thingsAPI.GET("", api.GetAllChores)
		thingsAPI.POST("/:id", api.PostCompleteChore)
	}

}
