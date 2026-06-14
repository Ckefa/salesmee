package business

import (
	"net/http"
	"salesmee/internal/models"
	"salesmee/internal/services"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func (h *BusinessHandler) ShowTeamLogin(c *gin.Context) {
	token, _ := c.Cookie("team_token")
	if token != "" {
		if claims, err := services.ValidateToken(token); err == nil && claims != nil {
			var member models.TeamMember
			if h.db.First(&member, claims.UserID).Error == nil && member.IsActive {
				c.Redirect(http.StatusFound, "/business/dashboard")
				return
			}
		}
	}
	activated := c.Query("activated") == "1"
	c.HTML(http.StatusOK, "team_login.html", gin.H{
		"Title":     "Staff Login - SalesMee",
		"Activated": activated,
	})
}

func (h *BusinessHandler) TeamLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var member models.TeamMember
	if err := h.db.Where("email = ? AND is_active = ?", req.Email, true).First(&member).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	if member.Password == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account not yet activated. Check your invite email."})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(member.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	now := time.Now()
	h.db.Model(&member).Update("last_login_at", &now)

	token, err := services.GenerateToken(member.ID, member.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session"})
		return
	}

	c.SetCookie("token", "", -1, "/business", "", false, true)
	c.SetCookie("team_token", token, 86400*7, "/", "", false, false)
	c.Set("team_member_id", member.ID)
	c.Set("business_id", member.BusinessID)
	c.Set("role", member.Role)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *BusinessHandler) TeamLogout(c *gin.Context) {
	c.SetCookie("team_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/business/team/login")
}
