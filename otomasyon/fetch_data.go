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

type Refactory struct {
	YmkTuru string `json:"ymk_turu"`
	Yil     string `json:"yil"`
	Gun     int    `json:"gun"`
	Ogle    string `json:"ogle"`
	Okalori string `json:"okalori"`
	Aksam   string `json:"aksam"`
	Akalori string `json:"akalori"`
}

type StudentDepartment struct {
	DepartmentName     string `json:"birimAdi"`
	DepartmentYear     string `json:"sinifi"`
	DepartmentSemester string `json:"donem"`
	StudentID          string `json:"ogrenciNo"`
}

type Profile struct {
	Name        string              `json:"ad"`
	Surname     string              `json:"soyad"`
	Nationality string              `json:"uyruk"`
	Departments []StudentDepartment `json:"ogrenciBirimBilgileriListe"`
}

type StudentBranch struct {
	DepartmentID   int    `json:"birimID"`
	DepartmentName string `json:"birimAdi"`
}

type Grade struct {
	CourseCode string `json:"dersKodu"`
	CourseName string `json:"dersAdi"`
	ECTS       string `json:"kredi"`
	Grade      string `json:"bNot"`
}

type SemesterGrades struct {
	SemesterID   int     `json:"donemID"`
	SemesterName string  `json:"donemAdi"`
	SemesterECTS string  `json:"yariYilKredi"`
	SemesterANO  string  `json:"yariYilAno"`
	TotalECTS    string  `json:"genelKredi"`
	GANO         string  `json:"genelAno"`
	Grades       []Grade `json:"ogrenciNotListe"`
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

func (u *UludagFetcher) sendRequest(endpoint string, student Student, alternativeURL ...string) (string, error) {
	url := "https://mobileservicev2.uludag.edu.tr/"
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
