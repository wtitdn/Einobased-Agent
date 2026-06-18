package handler

import (
	"errors"
	"net/http"

	request "einoproject/internal/controller/DTO/request"
	response "einoproject/internal/controller/DTO/response"
	"einoproject/internal/usecase"

	"github.com/gin-gonic/gin"
)

type AccountHandler struct {
	accountService *usecase.AccountService
}

func NewAccountHandler(accountService *usecase.AccountService) *AccountHandler {
	return &AccountHandler{accountService: accountService}
}
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var req request.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: err.Error()})
		return
	}

	account, err := h.accountService.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidAccountInput):
			c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: err.Error()})
		case errors.Is(err, usecase.ErrAccountExists):
			c.JSON(http.StatusConflict, response.ErrorResponse{Error: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, response.ErrorResponse{Error: err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, response.AccountResponse{
		ID:       account.ID,
		Username: account.Username,
	})
}

func (h *AccountHandler) Login(c *gin.Context) {
	var req request.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: err.Error()})
		return
	}

	result, err := h.accountService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidAccountInput):
			c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: err.Error()})
		case errors.Is(err, usecase.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, response.ErrorResponse{Error: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, response.ErrorResponse{Error: err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, response.LoginResponse{
		Account: response.AccountResponse{
			ID:       result.Account.ID,
			Username: result.Account.Username,
		},
		Token:        result.Token,
		RefreshToken: result.RefreshToken,
	})
}
