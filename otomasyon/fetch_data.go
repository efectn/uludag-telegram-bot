package otomasyon

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

type Student struct {
	StudentID           string `json:"studentID"`
	StudentSessionToken string `json:"studentSessionToken"`
}

type UludagFetcher struct {
	client *http.Client
}

type ExamResult struct {
	ExamID     int    `json:"sinavID"`
	ExamName   string `json:"sinavAdi"`
	ExamDate   string `json:"tarih"`
	ExamType   string `json:"sinavTipi"`
	ExamTypeID int    `json:"sinavTipiID"`
	ExamGrade  int    `json:"sinavNotu"`
}

type UserLoginSuccess struct {
	StudentID           string `json:"studentID"`
	StudentSessionToken string `json:"studentSessionToken"`
	Message             string `json:"message"`
}

func NewUludagFetcher() *UludagFetcher {
	return &UludagFetcher{
		client: &http.Client{
			Timeout: 20 * time.Second,
			Transport: &http.Transport{
				TLSHandshakeTimeout: 10 * time.Second,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // TLS sertifikası doğrulamasını devre dışı bırak
					MinVersion:         tls.VersionTLS12,
				},
			},
		},
	}
}

func (u *UludagFetcher) sendRequest(endpoint string, student Student) (string, error) {
	req, err := http.NewRequest("GET", "https://mobileservicev2.uludag.edu.tr/"+endpoint, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("User-Agent", "Dart/3.0 (dart:io)")
	req.Header.Set("student_session_token", student.StudentSessionToken)
	req.Header.Set("studentid", student.StudentID)

	resp, err := u.client.Do(req)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return string(body), nil
}

func (u *UludagFetcher) StudentLogin(studentid, password string) (string, bool, error) {
	req, err := http.NewRequest("POST", "https://mobileservicev2.uludag.edu.tr/login-student/studentlogin", nil)
	if err != nil {
		return "", false, err
	}

	req.Header.Set("User-Agent", "Dart/3.0 (dart:io)")
	req.Header.Set("loginname", studentid)
	req.Header.Set("password", password)

	resp, err := u.client.Do(req)
	if err != nil {
		return "", false, err
	}

	// Check if login is successful
	if resp.StatusCode != http.StatusOK {
		return "", false, nil
	}

	var user UserLoginSuccess

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, err
	}

	if err := json.Unmarshal(body, &user); err != nil {
		return "", false, err
	}

	if strings.Trim(user.Message, " ") == "Giriş Başarılı." {
		return user.StudentSessionToken, true, nil
	}

	return "", false, nil
}

func (u *UludagFetcher) CheckStudentToken(student Student) (bool, error) {
	body, err := u.sendRequest("login-student/studentchecktoken", student)
	if err != nil {
		return false, err
	}

	if strings.Trim(body, " ") != "\"Giriş Başarılı\"" {
		return false, nil
	}

	return true, nil
}

func (u *UludagFetcher) GetExamResults(student Student) ([]ExamResult, error) {
	body, err := u.sendRequest("student/examresults", student)
	if err != nil {
		return nil, err
	}

	var results []ExamResult
	if err := json.Unmarshal([]byte(body), &results); err != nil {
		return nil, err
	}

	return results, nil
}
