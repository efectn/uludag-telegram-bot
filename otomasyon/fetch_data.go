package otomasyon

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Student struct {
	StudentID           string `json:"studentID"`
	StudentSessionToken string `json:"studentSessionToken"`
	Branch              int    `json:"branch"`
}

type UludagFetcher struct {
	client *http.Client
}

const UludagMobileAPI = "https://mobileservicev2.uludag.edu.tr/"

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

func (u *UludagFetcher) sendRequest(endpoint string, student Student, alternativeURL ...string) (string, error) {
	url := UludagMobileAPI
	if len(alternativeURL) > 0 {
		url = alternativeURL[0]
	}

	req, err := http.NewRequest("GET", url+endpoint, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("User-Agent", "Dart/3.0 (dart:io)")
	req.Header.Set("student_session_token", student.StudentSessionToken)
	req.Header.Set("studentid", student.StudentID)

	if student.Branch != 0 {
		req.Header.Set("branch", strconv.Itoa(student.Branch))
	}

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
	req, err := http.NewRequest("POST", UludagMobileAPI+"login-student/studentlogin", nil)
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

func (u *UludagFetcher) GetStudentInfo(student Student) (Profile, error) {
	body, err := u.sendRequest("student/studentinfo", student)
	if err != nil {
		return Profile{}, err
	}

	var profile Profile
	if err := json.Unmarshal([]byte(body), &profile); err != nil {
		return Profile{}, err
	}

	return profile, nil
}

func (u *UludagFetcher) GetStudentBranches(student Student) ([]StudentBranch, error) {
	body, err := u.sendRequest("student/studentbranches", student)
	if err != nil {
		return nil, err
	}

	var branches []StudentBranch
	if err := json.Unmarshal([]byte(body), &branches); err != nil {
		return nil, err
	}

	return branches, nil
}

func (u *UludagFetcher) GetGradeCard(student Student) ([]SemesterGrades, error) {
	body, err := u.sendRequest("student/gradecard", student)
	if err != nil {
		return nil, err
	}

	var semesterGrades []SemesterGrades
	if err := json.Unmarshal([]byte(body), &semesterGrades); err != nil {
		return nil, err
	}

	return semesterGrades, nil
}

func (u *UludagFetcher) GetSyllabus(student Student) ([]SyllabusEntry, error) {
	body, err := u.sendRequest("student/syllabus", student)
	if err != nil {
		return nil, err
	}

	var syllabus []SyllabusEntry

	if err := json.Unmarshal([]byte(body), &syllabus); err != nil {
		return nil, err
	}

	return syllabus, nil
}

func (u *UludagFetcher) GetExamSchedule(student Student) ([]Exam, error) {
	body, err := u.sendRequest("student/examcalendar", student)
	if err != nil {
		return nil, err
	}

	var exams []Exam
	if err := json.Unmarshal([]byte(body), &exams); err != nil {
		return nil, err
	}

	return exams, nil
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

func (u *UludagFetcher) GetRefactoryList() (Refactory, error) {
	body, err := u.sendRequest("yemek/std", Student{}, "https://anasayfaws.uludag.edu.tr/")
	if err != nil {
		return Refactory{}, err
	}

	var refactory Refactory
	if err := json.Unmarshal([]byte(body), &refactory); err != nil {
		return Refactory{}, err
	}

	return refactory, nil
}
