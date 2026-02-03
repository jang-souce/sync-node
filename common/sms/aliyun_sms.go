package sms

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/dysmsapi"
)

// SMSClient 定义短信发送接口
type SMSClient interface {
	SendSMS(phoneNumbers []string, signName, templateCode string, templateParam map[string]string) error
}

// AliyunSMSClient 阿里云短信客户端实现
type AliyunSMSClient struct {
	client *dysmsapi.Client
}

// NewAliyunSMSClient 创建新的阿里云短信客户端
func NewAliyunSMSClient(accessKeyID, accessKeySecret string, regionID string) (*AliyunSMSClient, error) {
	if regionID == "" {
		regionID = "cn-hangzhou"
	}
	client, err := dysmsapi.NewClientWithAccessKey(regionID, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to init aliyun sms client: %w", err)
	}
	return &AliyunSMSClient{client: client}, nil
}

// SendSMS 发送短信
// phoneNumbers: 接收短信的手机号码列表
// signName: 短信签名名称
// templateCode: 短信模板Code
// templateParam: 模板参数
func (c *AliyunSMSClient) SendSMS(phoneNumbers []string, signName, templateCode string, templateParam map[string]string) error {
	if len(phoneNumbers) == 0 {
		return nil
	}

	request := dysmsapi.CreateSendSmsRequest()
	request.Scheme = "https"
	request.SignName = signName
	request.TemplateCode = templateCode

	if templateParam != nil {
		paramJSON, err := json.Marshal(templateParam)
		if err != nil {
			return fmt.Errorf("failed to marshal sms param: %w", err)
		}
		request.TemplateParam = string(paramJSON)
	}

	// 阿里云 SMS API 支持单个 PhoneNumbers 传多个号码(逗号分隔)，上限 1000
	request.PhoneNumbers = strings.Join(phoneNumbers, ",")

	response, err := c.client.SendSms(request)
	if err != nil {
		return fmt.Errorf("failed to send sms: %w", err)
	}
	if response.Code != "OK" {
		return fmt.Errorf("aliyun sms api error: %s - %s", response.Code, response.Message)
	}

	return nil
}
