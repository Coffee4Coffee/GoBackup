package scheduler

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/capnspacehook/taskmaster"
	"github.com/rickb777/date/period"
)

func TestParseTaskPathValid(t *testing.T) {
	testcases := []struct {
		src, dest, want string
	}{
		{`F:\GIMP\GIMP 2\share\mypaint-data\1.0\brushes\kaerhon_v1`, `E:\BackUp\Everything\試験\Test Spaces in path`, `F__GIMP_GIMP 2_share_mypaint-data_1.0_brushes_kaerhon_v1 E__BackUp_Everything_試験_Test Spaces in path`},
		{`C:\Users\UserName\AppData\Roaming\BKKKKKKKKKK\thumbnail`, `E:\BackupFolderForThisTest\.ssh`, `C__Users_UserName_AppData_Roaming_BKKKKKKKKKK_thumbnail E__BackupFolderForThisTest_.ssh`},
		{`C:\Windows\Microsoft.NET\assembly\GAC_MSIL\Microsoft.Transactions.Bridge.Dtc.resources\v4.0_4.0.0.0_ja_b03f5f7f11d50a3a`, `F:\UNREAL_Projects\GGGG\Saved\Cooked\WindowsNoEditor\Engine\Plugins\Runtime\Oculus\OculusVR\Content\Materials`, `C__Windows_Microsoft.NET_assembly_GAC_MSIL_Microsoft.Transactions.Bridge.Dtc.resources_v4.0_4.0.0.0_ja_b03f5f7f11d50a3a F__UNREAL_Projects_GGGG_Saved_Cooked_WindowsNoEditor_Engine_Plugins_Runtime_Oculus_OculusVR_Content_Materials`},
	}
	m := regexp.MustCompile(`[:<>\\/?*|"]`)
	for _, tc := range testcases {
		result := parseTaskPath(tc.src, tc.dest)
		if result != tc.want && m.MatchString(result) {
			t.Errorf(`parseTaskPath(src, dest) = %v, %v, want match for %v`, tc.src, tc.dest, result)
		}
	}
}

func TestParseTaskPathInvalid(t *testing.T) {
	testcases := []struct {
		src, dest, want string
	}{
		{`F:\GIMP\GIMP 2\share\mypaint-data\1.0\brushes\kaerhon_v1`, `E:\BackUp\Everything\試験\Test Spaces in path`, `F__GIMP_GIMP 2_share_mypaint-data_1<0_brushes_kaerhon_v1 E__BackUp_Everything_試験_Test Spaces in path`},
		{`C:\Users\UserName\AppData\Roaming\BKKKKKKKKKK\thumbnail`, `E:\BackupFolderForThisTest\.ssh`, `C__Users_UserName_AppData_Roaming_BKKKKKKKKKK_thumbnail E__BackupFolderForThisTest_.ssh`},
		{`C:\Windows\Microsoft.NET\assembly\GAC_MSIL\Microsoft.Transactions.Bridge.Dtc.resources\:v4.0_4.0.0.0_ja_b03f5f7f11d50a3a`, `F:\UNREAL_Projects\GGGG\Saved\Cooked\WindowsNoEditor\Engine\Plugins\Runtime\Oculus\OculusVR\Content\Materials`, `C__Windows_Microsoft.NET_assembly_GAC_MSIL_Microsoft.Transactions*.Bridge.Dtc.resources_v4.0_4.0.0.0_ja_b03f5f7f11d50a3a F__UNREAL_Projects_GGGG_Saved_Cooked_WindowsNoEditor_Engine_Plugins_Runtime_Oculus_OculusVR_Content_Materials`},
	}
	m := regexp.MustCompile(`[:<>\\/?*|"]`)
	for _, tc := range testcases {
		result := parseTaskPath(tc.src, tc.dest)
		// Test if the output contains illegal characters
		if result == tc.want && m.MatchString(result) {
			t.Errorf(`parseTaskPath(src, dest) = %v, %v, want match for %v`, tc.src, tc.dest, result)
		}
	}
}

func FuzzParseTaskPath(f *testing.F) {
	testcases := []struct {
		src, dest string
	}{
		{`F:\GIMP\GIMP 2\share\mypaint-data\1.0\brushes\kaerhon_v1`, `E:\BackUp\Everything\試験\Test Spaces in path`},
		{`C:\Users\UserName\AppData\Roaming\BKKKKKKKKKK\thumbnail`, `E:\BackupFolderForThisTest\.ssh`},
		{`C:\Windows\Microsoft.NET\assembly\GAC_MSIL\Microsoft.Transactions.Bridge.Dtc.resources\:v4.0_4.0.0.0_ja_b03f5f7f11d50a3a`, `F:\UNREAL_Projects\GGGG\Saved\Cooked\WindowsNoEditor\Engine\Plugins\Runtime\Oculus\OculusVR\Content\Materials`},
	}
	for _, tc := range testcases {
		f.Add(tc.src, tc.dest)
	}

	f.Fuzz(func(t *testing.T, src, dest string) {
		result := parseTaskPath(src, dest)
		m := regexp.MustCompile(`[:<>\\/?*|"]`)

		t.Logf("Input: src=%v\n dest=%v\n result=%v", src, dest, result)
		if m.MatchString(result) {
			t.Errorf("Parsed task path: %v, contains illegal character", result)
		}
	})
}

func TestGetValidTime(t *testing.T) {
	tNow := time.Now()
	// 2x test for day+0, 2x tests for day+1, 2x tests for invalid numbers
	testcases := []uint8{
		uint8((12 % tNow.Hour())),
		uint8((19 % tNow.Hour())),
		uint8((6 % (23 - tNow.Hour())) + tNow.Hour()),
		uint8((13 % (23 - tNow.Hour())) + tNow.Hour()),
		uint8(24),
		uint8(125),
	}
	for _, dHour := range testcases {
		result, err := getValidTime(dHour)
		if dHour > 23 && err != nil {
			return
		}
		if dHour > 23 && err == nil {
			t.Errorf(`getValidTime(dHour) = %v, invalid hour error not returned`, dHour)
		}
		if result.Hour() != int(dHour) {
			t.Errorf(`getValidTime(dHour) = %v, want Hour match for %v`, dHour, result.Hour())
		}
		if int(dHour) < tNow.Hour() && result.Day() == tNow.Day() {
			t.Errorf(`getValidTime(dHour) = %v, want Day match for %v`, tNow.Day(), result.Day())
		}
		if int(dHour) > tNow.Hour() && result.Day() != tNow.Day() {
			t.Errorf(`getValidTime(dHour) = %v, want Day match for %v`, tNow.Day(), result.Day())
		}
	}
}

func FuzzGetValidTime(f *testing.F) {
	tNow := time.Now()
	testcases := []uint8{0, 1, 6, 12, 17, 23, 25}
	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, dHour uint8) {
		result, err := getValidTime(dHour)

		t.Logf("Input: dHour=%v \n result Hour=%v\n result Day=%v\n error=%v", dHour, result.Hour(), result.Day(), err)
		if dHour > 23 && err != nil {
			return
		}
		if dHour > 23 && err == nil {
			t.Errorf(`getValidTime(dHour) = %v, invalid hour error not returned`, dHour)
		}
		if result.Hour() != int(dHour) {
			t.Errorf(`getValidTime(dHour) = %v, want Hour match for %v`, dHour, result.Hour())
		}
		if int(dHour) < tNow.Hour() && result.Day() == tNow.Day() {
			t.Errorf(`getValidTime(dHour) = %v, want Day match for %v`, tNow.Day(), result.Day())
		}
		if int(dHour) > tNow.Hour() && result.Day() != tNow.Day() {
			t.Errorf(`getValidTime(dHour) = %v, want Day match for %v`, tNow.Day(), result.Day())
		}
	})
}

func TestCreateTrigger(t *testing.T) {
	startDate, _ := getValidTime(12)
	testcases := []struct {
		tType                TriggerType
		dMonth, dWeek, dHour uint8
		wantTrigger          taskmaster.Trigger
		wantError            error
	}{
		{
			TriggerType(0), 0, 0, 12, taskmaster.DailyTrigger{
				TaskTrigger: taskmaster.TaskTrigger{
					Enabled:       true,
					StartBoundary: startDate,
					RepetitionPattern: taskmaster.RepetitionPattern{
						RepetitionDuration: period.NewYMD(0, 0, 365),
						RepetitionInterval: period.NewHMS(24, 0, 0),
					},
				},
				DayInterval: taskmaster.EveryDay,
			}, nil,
		},
		{
			TriggerType(1), 0, 4, 0, taskmaster.WeeklyTrigger{
				TaskTrigger: taskmaster.TaskTrigger{
					Enabled:       true,
					StartBoundary: startDate,
					RepetitionPattern: taskmaster.RepetitionPattern{
						RepetitionDuration: period.NewYMD(0, 0, 365),
					},
				},
				WeekInterval: taskmaster.EveryWeek,
				DaysOfWeek:   taskmaster.Wednesday,
			}, nil,
		},
		{
			TriggerType(2), 6, 0, 0, taskmaster.MonthlyTrigger{
				TaskTrigger: taskmaster.TaskTrigger{
					Enabled:       true,
					StartBoundary: startDate,
					RepetitionPattern: taskmaster.RepetitionPattern{
						RepetitionDuration: period.NewYMD(0, 0, 365),
					},
				},
				DaysOfMonth:  taskmaster.Six,
				MonthsOfYear: taskmaster.AllMonths,
			}, nil,
		},
		{
			TriggerType(2), 31, 0, 0, nil, fmt.Errorf("createTrigger: %w", errors.New("invalid day of month/week")),
		},
		{
			TriggerType(2), 30, 7, 0, nil, fmt.Errorf("createTrigger: %w", errors.New("invalid day of month/week")),
		},
		{
			TriggerType(1), 12, 4, 25, nil, fmt.Errorf("createTrigger: %w", fmt.Errorf("getValidTime: %w", errors.New("invalid hour"))),
		},
	}
	for _, tc := range testcases {
		result, err := createTrigger(tc.tType, tc.dMonth, tc.dWeek, tc.dHour)

		if result != tc.wantTrigger || err != tc.wantError {
			t.Errorf(`createTrigger(tc.tType, tc.dMonth, tc.dWeek, tc.dHour) = %v, %v, %v, %v want match for value: %v, error: %v`, tc.tType, tc.dMonth, tc.dWeek, tc.dHour, tc.wantTrigger, tc.wantError)
		}
	}
}

func TestCreateAction(t *testing.T) {
	testcases := []struct {
		src, dest   string
		backupLimit uint8
		overwrite   bool
		wantAction  taskmaster.ExecAction
		wantError   error
	}{
		{`C:\test`, `Z:\backupme`, 0, false, taskmaster.ExecAction{}, fmt.Errorf("createAction: failed to retrieve systemdrive: %w", errors.New("SYSTEMDRIVE not found"))},
	}

	oldSysDrive := os.Getenv("SYSTEMDRIVE")
	os.Setenv("SYSTEMDRIVE", "")
	for _, tc := range testcases {
		result, err := createAction(tc.src, tc.dest, tc.backupLimit, tc.overwrite)

		if result != tc.wantAction && err != tc.wantError {
			os.Setenv("SYSTEMDRIVE", oldSysDrive)
			t.Errorf(`createAction(...) = %v, want match for %v; err: %v, want match for %v`, result, tc.wantAction, err, tc.wantError)
		}
	}
	os.Setenv("SYSTEMDRIVE", oldSysDrive)
}
