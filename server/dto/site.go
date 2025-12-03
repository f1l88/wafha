package dto

import (
	"time"

	"github.com/HUAHUAI23/RuiQi/server/model"
)

// CreateSiteRequest Create a site request
// @Description Request parameters for creating a site
type CreateSiteRequest struct {
	Name         string          `json:"name" binding:"required" example:"my-site"`                                      // Site name
	Domain       string          `json:"domain" binding:"required,domain" example:"example.com"`                         // domain name
	ListenPort   int             `json:"listenPort" binding:"required,min=1,max=65535" example:"8080"`                   // Monitor port
	EnableHTTPS  bool            `json:"enableHTTPS" example:"false"`                                                    // Enable HTTPS
	Certificate  *CertificateDTO `json:"certificate,omitempty" binding:"omitempty,required_if=EnableHTTPS true"`         // Certificate information
	Backend      BackendDTO      `json:"backend" binding:"required"`                                                     // Back-end server configuration
	WAFEnabled   bool            `json:"wafEnabled" example:"false"`                                                     // Enable WAF
	WAFMode      string          `json:"wafMode" binding:"omitempty,oneof=protection observation" example:"observation"` // WAF mode
	ActiveStatus bool            `json:"activeStatus" example:"true"`                                                    // Site status
}

// UpdateSiteRequest Update site request
// @Description Update the request parameters of the site
type UpdateSiteRequest struct {
	Name         string          `json:"name,omitempty" binding:"omitempty" example:"my-site"`                           // 站点名称
	Domain       string          `json:"domain,omitempty" binding:"omitempty,domain" example:"example.com"`              // 域名
	ListenPort   int             `json:"listenPort,omitempty" binding:"omitempty,min=1,max=65535" example:"8080"`        // 监听端口
	EnableHTTPS  bool            `json:"enableHTTPS" example:"false"`                                                    // 是否启用HTTPS
	Certificate  *CertificateDTO `json:"certificate,omitempty" binding:"omitempty,required_if=EnableHTTPS true"`         // 证书信息
	Backend      *BackendDTO     `json:"backend,omitempty" binding:"omitempty"`                                          // 后端服务器配置
	WAFEnabled   bool            `json:"wafEnabled" example:"false"`                                                     // 是否启用WAF
	WAFMode      string          `json:"wafMode" binding:"omitempty,oneof=protection observation" example:"observation"` // WAF模式
	ActiveStatus bool            `json:"activeStatus" example:"true"`                                                    // 站点状态
}

// CertificateDTO 证书DTO
type CertificateDTO struct {
	CertName    string    `json:"certName" binding:"required" example:"my-cert"`         // 证书名称
	PublicKey   string    `json:"publicKey" binding:"required"`                          // 公钥内容
	PrivateKey  string    `json:"privateKey" binding:"required"`                         // 私钥内容
	ExpireDate  time.Time `json:"expireDate" binding:"required"`                         // 过期时间
	IssuerName  string    `json:"issuerName" binding:"required" example:"Let's Encrypt"` // 颁发机构
	FingerPrint string    `json:"fingerPrint" binding:"required"`                        // 证书指纹
}

// BackendDTO 后端服务器配置DTO
type BackendDTO struct {
	Servers []ServerDTO `json:"servers" binding:"required,min=1,dive"` // 服务器列表，至少需要一个服务器
}

// ServerDTO 服务器DTO
type ServerDTO struct {
	Host  string `json:"host" binding:"required" example:"backend.example.com"` // 主机地址
	Port  int    `json:"port" binding:"required,min=1,max=65535" example:"80"`  // 端口
	IsSSL bool   `json:"isSSL" example:"false"`                                 // 是否启用SSL
}

// SiteResponse 站点响应
// @Description 站点信息响应
type SiteResponse struct {
	model.Site
}

// SiteListResponse 站点列表响应
// @Description 站点列表响应
type SiteListResponse struct {
	Total int64        `json:"total"` // 总数
	Items []model.Site `json:"items"` // 站点列表
}
