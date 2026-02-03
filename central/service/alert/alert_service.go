package alert

import (
	"sync-node/central/config"
	"sync-node/common/sms"
	"sync-node/common/utils"
)

// AlertService 定义报警服务接口
type AlertService interface {
	SendAlert(param map[string]string) error
}

// AliyunSMSAlertService 阿里云短信报警服务实现
type AliyunSMSAlertService struct {
	client sms.SMSClient
	cfg    *config.Config
}

// NewAlertService 创建报警服务实例
func NewAlertService(cfg *config.Config) (*AliyunSMSAlertService, error) {
	client, err := sms.NewAliyunSMSClient(cfg.SMS.AccessKeyID, cfg.SMS.AccessKeySecret, "cn-hangzhou")
	if err != nil {
		return nil, err
	}
	return &AliyunSMSAlertService{client: client, cfg: cfg}, nil
}

// SendAlert 发送报警信息
func (s *AliyunSMSAlertService) SendAlert(param map[string]string) error {
	// 如果未配置接收号码，仅记录日志
	if len(s.cfg.SMS.PhoneNumbers) == 0 {
		utils.GetLogger("alert").Warn("No phone numbers configured for alerts")
		return nil
	}

	// 触发告警，发送短信通知（告警类型、节点ID、任务ID、错误信息、触发时间）
	err := s.client.SendSMS(s.cfg.SMS.PhoneNumbers, s.cfg.SMS.SignName, s.cfg.SMS.TemplateCode, param)
	if err != nil {
		return err
	}

	utils.GetLogger("alert").Infof("Alert sent successfully to %v", s.cfg.SMS.PhoneNumbers)
	return nil
}
