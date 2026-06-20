package business

import (
	"net/http"
	"salesmee/internal/models"
	"salesmee/internal/services"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func (h *TeamHandler) ShowTeamLogin(c *gin.Context) {
	token, _ := c.Cookie("team_token")
	if token != "" {
		if claims, err := services.ValidateToken(token); err == nil && claims != nil {
			var member models.TeamMember
			if h.dbc(c).First(&member, claims.UserID).Error == nil && member.IsActive {
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

func (h *TeamHandler) TeamLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var member models.TeamMember
	if err := h.dbc(c).Where("email = ? AND is_active = ?", req.Email, true).First(&member).Error; err != nil {
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
	h.dbc(c).Model(&member).Update("last_login_at", &now)

	token, err := services.GenerateToken(member.ID, member.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session"})
		return
	}

	services.ClearCookie(c, "token", "/business")
	services.SetSecureCookie(c, "team_token", token, 86400*7, "/")
	c.Set("team_member_id", member.ID)
	c.Set("business_id", member.BusinessID)
	c.Set("role", member.Role)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *TeamHandler) TeamLogout(c *gin.Context) {
	services.ClearCookie(c, "team_token", "/")
	c.Redirect(http.StatusFound, "/business/team/login")
}
