package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/msojocs/free2api/server/internal/model"
	"github.com/msojocs/free2api/server/internal/repository"
	"github.com/msojocs/free2api/server/pkg/crypto"
	"gorm.io/gorm"
)

type PushTemplateService struct {
	repo        repository.PushTemplateRepository
	accountRepo repository.AccountRepository
}

func NewPushTemplateService(repo repository.PushTemplateRepository, accountRepo repository.AccountRepository) *PushTemplateService {
	return &PushTemplateService{repo: repo, accountRepo: accountRepo}
}

type CreatePushTemplateRequest struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	Method       string `json:"method"`
	Headers      string `json:"headers"`
	QueryParams  string `json:"query_params"`
	BodyTemplate string `json:"body_template"`
	Description  string `json:"description"`
	AccountType  string `json:"account_type"`
}

type UpdatePushTemplateRequest struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	Method       string `json:"method"`
	Headers      string `json:"headers"`
	QueryParams  string `json:"query_params"`
	BodyTemplate string `json:"body_template"`
	Description  string `json:"description"`
	Enabled      bool   `json:"enabled"`
	AccountType  string `json:"account_type"`
}

func (s *PushTemplateService) GetTemplate(id uint) (*model.PushTemplate, error) {
	return s.repo.FindByID(id)
}

func (s *PushTemplateService) List(page, limit int) ([]model.PushTemplate, int64, error) {
	offset := (page - 1) * limit
	return s.repo.List(offset, limit)
}

func (s *PushTemplateService) ListEnabledForType(accountType string) ([]model.PushTemplate, error) {
	return s.repo.ListEnabledByType(accountType)
}

func (s *PushTemplateService) Create(req CreatePushTemplateRequest) (*model.PushTemplate, error) {
	method := strings.ToUpper(req.Method)
	if method == "" {
		method = "POST"
	}
	t := &model.PushTemplate{
		Name:         req.Name,
		URL:          req.URL,
		Method:       method,
		Headers:      req.Headers,
		QueryParams:  req.QueryParams,
		BodyTemplate: req.BodyTemplate,
		Description:  req.Description,
		AccountType:  req.AccountType,
		Enabled:      true,
		IsSystem:     false,
	}
	if err := s.repo.Create(t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *PushTemplateService) Update(id uint, req UpdatePushTemplateRequest) (*model.PushTemplate, error) {
	t, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	method := strings.ToUpper(req.Method)
	if method == "" {
		method = t.Method
	}
	t.Name = req.Name
	t.URL = req.URL
	t.Method = method
	t.Headers = req.Headers
	t.QueryParams = req.QueryParams
	t.BodyTemplate = req.BodyTemplate
	t.Description = req.Description
	t.Enabled = req.Enabled
	t.AccountType = req.AccountType
	if err := s.repo.Update(t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *PushTemplateService) Delete(id uint) error {
	t, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if t.IsSystem {
		return errors.New("system templates cannot be deleted")
	}
	return s.repo.Delete(id)
}

func (s *PushTemplateService) CopyTemplate(id uint) (*model.PushTemplate, error) {
	src, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	copyT := &model.PushTemplate{
		Name:         "Copy of " + src.Name,
		Enabled:      src.Enabled,
		URL:          src.URL,
		Method:       src.Method,
		Headers:      src.Headers,
		BodyTemplate: src.BodyTemplate,
		Description:  src.Description,
		AccountType:  src.AccountType,
		IsSystem:     false,
	}
	if err := s.repo.Create(copyT); err != nil {
		return nil, err
	}
	return copyT, nil
}

// PushAccount fires async HTTP pushes to all enabled templates whose AccountType matches
// the account's type (or templates with an empty AccountType).
func (s *PushTemplateService) PushAccount(account *model.Account) {
	go func() {
		templates, err := s.repo.ListEnabledByType(account.Type)
		if err != nil {
			log.Printf("[push_template] failed to list enabled templates: %v", err)
			return
		}

		plainPassword, err := crypto.Decrypt(account.Password)
		if err != nil {
			log.Printf("[push_template] failed to decrypt password for account %d: %v", account.ID, err)
			plainPassword = ""
		}

		data := map[string]interface{}{
			"email":      account.Email,
			"password":   plainPassword,
			"type":       account.Type,
			"status":     account.Status,
			"extra":      account.Extra,
			"task_id":    account.TaskBatchID,
			"created_at": account.CreatedAt.UTC().Format(time.RFC3339),
		}

		client := &http.Client{Timeout: 10 * time.Second}
		for _, tmpl := range templates {
			if err := executePush(client, tmpl, data); err != nil {
				log.Printf("[push_template] template %q (id=%d) push failed: %v", tmpl.Name, tmpl.ID, err)
			}
		}
	}()
}

// PushAccountByID manually pushes a specific account to a specific template.
// Returns the HTTP status code and response body (or an error).
func (s *PushTemplateService) PushAccountByID(accountID, templateID uint) (int, string, error) {
	account, err := s.accountRepo.FindByID(accountID)
	if err != nil {
		return 0, "", err
	}

	tmpl, err := s.repo.FindByID(templateID)
	if err != nil {
		return 0, "", err
	}

	plainPassword, _ := crypto.Decrypt(account.Password)

	data := map[string]interface{}{
		"email":      account.Email,
		"password":   plainPassword,
		"type":       account.Type,
		"status":     account.Status,
		"extra":      account.Extra,
		"task_id":    account.TaskBatchID,
		"created_at": account.CreatedAt.UTC().Format(time.RFC3339),
	}

	rendered, err := renderTemplate(tmpl.BodyTemplate, data)
	if err != nil {
		return 0, "", err
	}

	method := strings.ToUpper(tmpl.Method)
	var body io.Reader
	if method != "GET" {
		body = strings.NewReader(rendered)
	}

	targetURL := tmpl.URL
	if tmpl.QueryParams != "" {
		var rawParams map[string]string
		if jsonErr := json.Unmarshal([]byte(tmpl.QueryParams), &rawParams); jsonErr == nil {
			if parsedURL, urlErr := url.Parse(targetURL); urlErr == nil {
				q := parsedURL.Query()
				for k, v := range rawParams {
					if rv, rErr := renderTemplate(v, data); rErr == nil {
						q.Set(k, rv)
					} else {
						q.Set(k, v)
					}
				}
				parsedURL.RawQuery = q.Encode()
				targetURL = parsedURL.String()
			}
		}
	}

	req, err := http.NewRequest(method, targetURL, body)
	if err != nil {
		return 0, "", err
	}

	if tmpl.Headers != "" {
		if headers, parseErr := parseHeaders(tmpl.Headers); parseErr == nil {
			for k, v := range headers {
				req.Header.Set(k, v)
			}
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(respBytes), nil
}

func executePush(client *http.Client, tmpl model.PushTemplate, data map[string]interface{}) error {
	rendered, err := renderTemplate(tmpl.BodyTemplate, data)
	if err != nil {
		return err
	}

	var body io.Reader
	method := strings.ToUpper(tmpl.Method)
	if method != "GET" {
		body = strings.NewReader(rendered)
	}

	// Append query parameters (also support Go template variables in values).
	targetURL := tmpl.URL
	if tmpl.QueryParams != "" {
		var rawParams map[string]string
		if jsonErr := json.Unmarshal([]byte(tmpl.QueryParams), &rawParams); jsonErr == nil {
			parsedURL, urlErr := url.Parse(targetURL)
			if urlErr == nil {
				q := parsedURL.Query()
				for k, v := range rawParams {
					if rendered, rErr := renderTemplate(v, data); rErr == nil {
						q.Set(k, rendered)
					} else {
						q.Set(k, v)
					}
				}
				parsedURL.RawQuery = q.Encode()
				targetURL = parsedURL.String()
			}
		}
	}

	req, err := http.NewRequest(method, targetURL, body)
	if err != nil {
		return err
	}

	if tmpl.Headers != "" {
		headers, parseErr := parseHeaders(tmpl.Headers)
		if parseErr != nil {
			log.Printf("[push_template] failed to parse headers: %v", parseErr)
		} else {
			for k, v := range headers {
				req.Header.Set(k, v)
			}
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return errors.New("push returned HTTP " + http.StatusText(resp.StatusCode) + ": " + string(respBody))
	}
	return nil
}

func renderTemplate(bodyTmpl string, data map[string]interface{}) (string, error) {
	t, err := template.New("body").Parse(bodyTmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func parseHeaders(headersJSON string) (map[string]string, error) {
	var m map[string]string
	if err := json.Unmarshal([]byte(headersJSON), &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *PushTemplateService) SeedSystemTemplates() {
	templates, _, err := s.repo.List(0, 1000)
	if err != nil {
		log.Printf("[push_template] SeedSystemTemplates: failed to list templates: %v", err)
		return
	}
	for _, t := range templates {
		if t.IsSystem {
			return
		}
	}

	cpa := &model.PushTemplate{
		Name:         "CLIProxyAPI (CPA)",
		Enabled:      false,
		URL:          "http://localhost:3000/api/accounts",
		Method:       "POST",
		Headers:      `{"Content-Type": "application/json"}`,
		BodyTemplate: `{"type": "{{.type}}", "email": "{{.email}}", "password": "{{.password}}", "token": "{{.extra}}"}`,
		Description:  "Built-in CPA (CLIProxyAPI) push template. Set the URL to your CLIProxyAPI instance and enable it.",
		IsSystem:     true,
		AccountType:  "",
	}
	if err := s.repo.Create(cpa); err != nil {
		log.Printf("[push_template] SeedSystemTemplates: failed to seed CPA template: %v", err)
	}
}

func (s *PushTemplateService) RegisterDBHook(db *gorm.DB) {
	db.Callback().Create().After("gorm:create").Register("push_template:after_create_account", func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Schema == nil {
			return
		}
		if tx.Statement.Schema.Name != "accounts" {
			return
		}
		account, ok := tx.Statement.Model.(*model.Account)
		if !ok {
			return
		}
		s.PushAccount(account)
	})
}
