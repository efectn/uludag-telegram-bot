package otomasyon

type ExamResult struct {
	ExamName   string  `json:"sinavAdi"`
	ExamDate   string  `json:"tarih"`
	ExamType   string  `json:"sinavTipi"`
	ExamID     int     `json:"sinavID"`
	ExamTypeID int     `json:"sinavTipiID"`
	ExamGrade  float64 `json:"sinavNotu"`
}

type UserLoginSuccess struct {
	StudentID           string `json:"studentID"`
	StudentSessionToken string `json:"studentSessionToken"`
	Message             string `json:"message"`
}

type Refactory struct {
	YmkTuru string `json:"ymk_turu"`
	Yil     string `json:"yil"`
	Ogle    string `json:"ogle"`
	Okalori string `json:"okalori"`
	Aksam   string `json:"aksam"`
	Akalori string `json:"akalori"`
	Gun     int    `json:"gun"`
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
	DepartmentName string `json:"birimAdi"`
	DepartmentID   int    `json:"birimID"`
}

type Grade struct {
	CourseCode string `json:"dersKodu"`
	CourseName string `json:"dersAdi"`
	ECTS       string `json:"kredi"`
	Grade      string `json:"bNot"`
}

type SemesterGrades struct {
	SemesterName string  `json:"donemAdi"`
	SemesterECTS string  `json:"yariYilKredi"`
	SemesterANO  string  `json:"yariYilAno"`
	TotalECTS    string  `json:"genelKredi"`
	GANO         string  `json:"genelAno"`
	Grades       []Grade `json:"ogrenciNotListe"`
	SemesterID   int     `json:"donemID"`
}

type SyllabusEntry struct {
	CourseCode string `json:"dersKodu"`
	Hours      string `json:"saatler"`
	ClassCode  string `json:"derslikKodu"`
	Day        int    `json:"gun"`
	Exists     int    `json:"dolu"`
}

type Exam struct {
	ExamName     string `json:"sinavAdi"`
	ExamDate     string `json:"tarih"`
	ExamType     string `json:"sinavTipi"`
	ExamTime     string `json:"sinavSaati"`
	ExamDuration string `json:"sinavSure"`
	ExamTypeID   int    `json:"sinavTipiID"`
}
