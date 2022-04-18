package scheduler

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/rickb777/date/period"

	"github.com/capnspacehook/taskmaster"
)

type TriggerType uint8

const (
	daily TriggerType = iota
	weekly
	monthly
)
const fPath = "\\GoBackup"

const (
	toastTemplate                      = "ToastText02"
	appTitle                           = "GoBackup"
	toastExpirationTimeInMinutes uint8 = 5
)

func parseTaskPath(src, dest string) string {
	m := regexp.MustCompile(`[:<>\\/?*|"]`)
	replaceWith := "_"
	src = m.ReplaceAllString(src, replaceWith)
	dest = m.ReplaceAllString(dest, replaceWith)
	return src + ` ` + dest
}

func getValidTime(dHour uint8) (time.Time, error) {
	if dHour > 23 {
		return time.Now(), fmt.Errorf("getValidTime: %w", errors.New("invalid hour"))
	}
	t := time.Now()
	if t.Hour() < int(dHour) {
		return time.Date(t.Year(), t.Month(), t.Day(), int(dHour), 0, 0, t.Nanosecond(), t.Location()), nil
	}
	// Return time for the next day if already in the past
	return time.Date(t.Year(), t.Month(), t.Day(), int(dHour), 0, 0, t.Nanosecond(), t.Location()).Add(24 * time.Hour), nil
}

// TODO: Change to robocopy down the line
func createToastScript(src, dest string) string {
	toastyString := fmt.Sprintf(`
		$ErrorActionPreference = 'Stop';
		[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null; 
		$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::%v); 
		$toastXml = [xml] $template.GetXml(); $toastXml.GetElementsByTagName('text')[0].AppendChild($toastXml.CreateTextNode($notificationTitle)) > $null; 
		$toastXml.GetElementsByTagName('text')[1].AppendChild($toastXml.CreateTextNode($notificationContent)) > $null; 
		$actionsElement = $toastXml.CreateElement('actions'); $actionElement = $toastXml.CreateElement('action'); 
		$actionElement.SetAttribute('content', 'Dismiss'); $actionElement.SetAttribute('arguments', 'dismiss'); 
		$actionElement.SetAttribute('activationType', 'system'); 
		$actionsElement.AppendChild($actionElement); 
		$toastXml.DocumentElement.AppendChild($actionsElement); 
		$xml = New-Object Windows.Data.Xml.Dom.XmlDocument; 
		$xml.LoadXml($toastXml.OuterXml); 
		$toast = [Windows.UI.Notifications.ToastNotification]::new($xml); 
		$toast.Tag = '%[2]v'; 
		$toast.Group = '%[2]v'; 
		$toast.ExpirationTime = [DateTimeOffset]::Now.AddMinutes(%[3]v); 
		if($actioncentre) { $toast.SuppressPopup = $true; };
		$notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('%[2]v'); 
		$notifier.Show($toast);
	`, toastTemplate, appTitle, toastExpirationTimeInMinutes)
	return fmt.Sprintf(`
		if ( $LASTEXITCODE -EQ 0 ) {
			$notificationTitle = [DateTime]::Now.ToShortTimeString() + ': ' + 'Your scheduled backup was successful'; 
			$notificationContent = 'You folder %[1]v has been backed up to %[2]v'; 
			`+toastyString+`
		};
		if( $LASTEXITCODE -EQ 1 ) {
			$notificationTitle = [DateTime]::Now.ToShortTimeString() + ': ' + 'Your scheduled backup has failed'; 
			$notificationContent = 'You folder %[1]v has not been backed up to %[2]v. No files were found to copy.'; 
			`+toastyString+`
		};
		if( $LASTEXITCODE -EQ 4 ) {
			$notificationTitle = [DateTime]::Now.ToShortTimeString() + ': ' + 'Your scheduled backup has failed'; 
			$notificationContent = 'You folder %[1]v has not been backed up to %[2]v. There is not enough memory or disk space.'; 
			`+toastyString+`
		};
		if( $LASTEXITCODE -EQ 5 ) {
			$notificationTitle = [DateTime]::Now.ToShortTimeString() + ': ' + 'Your scheduled backup has failed'; 
			$notificationContent = 'You folder %[1]v has not been backed up to %[2]v. A disk write error occurred.'; 
			`+toastyString+`
		};
	 `, src, dest)
}

func createTrigger(tType TriggerType, dMonth, dWeek, dHour uint8) (taskmaster.Trigger, error) {
	// RepetitionDuration set to 365 days as a workaround to incorrect parsing of period in go-ole
	// https://github.com/capnspacehook/taskmaster/issues/15
	startDate, err := getValidTime(dHour)
	if err != nil {
		return nil, fmt.Errorf("createTrigger: %w", err)
	}
	if dMonth > 30 || dWeek > 6 {
		return nil, fmt.Errorf("createTrigger: %w", errors.New("invalid day of month/week"))
	}
	if tType == daily {
		return taskmaster.DailyTrigger{
			TaskTrigger: taskmaster.TaskTrigger{
				Enabled:       true,
				StartBoundary: startDate,
				RepetitionPattern: taskmaster.RepetitionPattern{
					RepetitionDuration: period.NewYMD(0, 0, 365),
					RepetitionInterval: period.NewHMS(24, 0, 0),
				},
			},
			DayInterval: taskmaster.EveryDay,
		}, nil
	} else if tType == weekly {
		return taskmaster.WeeklyTrigger{
			TaskTrigger: taskmaster.TaskTrigger{
				Enabled:       true,
				StartBoundary: startDate,
				RepetitionPattern: taskmaster.RepetitionPattern{
					RepetitionDuration: period.NewYMD(0, 0, 365),
				},
			},
			WeekInterval: taskmaster.EveryWeek,
			DaysOfWeek:   taskmaster.DayOfWeek(1 << dWeek),
		}, nil
	} else {
		return taskmaster.MonthlyTrigger{
			TaskTrigger: taskmaster.TaskTrigger{
				Enabled:       true,
				StartBoundary: startDate,
				RepetitionPattern: taskmaster.RepetitionPattern{
					RepetitionDuration: period.NewYMD(0, 0, 365),
				},
			},
			DaysOfMonth:  taskmaster.DayOfMonth(1 << dMonth),
			MonthsOfYear: taskmaster.AllMonths,
		}, nil
	}
}

func createAction(src, dest string, overwrite bool) (taskmaster.ExecAction, error) {
	r := regexp.MustCompile(`[^\\]+$`)
	folder := r.FindString(src)

	// pwsPath := `\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`
	systemDrive := os.Getenv("SYSTEMDRIVE")
	toastCmd := createToastScript(src, dest)
	if len(systemDrive) == 0 {
		return taskmaster.ExecAction{}, fmt.Errorf("createAction: failed to retrieve systemdrive: %w", errors.New("SYSTEMDRIVE not found"))
	}
	xcopyCmd := fmt.Sprintf(`xcopy %v %v\%v /e /y /i /h /o /k /x;`, src, dest, folder)
	if !overwrite {
		renameCmd := `cd ` + dest + `; ` + `ren ` + folder + ` "` + folder + ` %date:~0,4%%date:~5,2%%date:~8,2%_%time:~0,2%%time:~3,2%%time:~6,2%` + `";`
		return taskmaster.ExecAction{
			Path: `Powershell`,
			Args: xcopyCmd + toastCmd + renameCmd,
		}, nil
	} else {
		return taskmaster.ExecAction{
			Path: `Powershell`,
			Args: xcopyCmd + toastCmd,
		}, nil
	}
}

func GetAllScheduledTasks() (taskmaster.RegisteredTaskCollection, error) {
	conn, err := taskmaster.Connect()
	if err != nil {
		return taskmaster.RegisteredTaskCollection{}, &ErrConnectSchedulerFailure{Inner: err, Message: "failed to connect to task scheduler"}
	}
	defer conn.Disconnect()

	tFolder, err := conn.GetTaskFolder(fPath)
	if err != nil {
		return taskmaster.RegisteredTaskCollection{}, &ErrRetrieveTaskFolderFailure{Inner: err, Message: "failed to find task folder"}
	}

	return tFolder.RegisteredTasks, nil
}

func CreateScheduledTask(tType TriggerType, dMonth, dWeek, dHour uint8, src, dest string, overwrite bool) (taskmaster.RegisteredTask, error) {
	conn, err := taskmaster.Connect()
	if err != nil {
		return taskmaster.RegisteredTask{}, err
	}
	defer conn.Disconnect()

	def := conn.NewTaskDefinition()

	trigger, err := createTrigger(tType, dMonth, dWeek, dHour)
	if err != nil {
		return taskmaster.RegisteredTask{}, &ErrCreateTaskFailure{Inner: err, Message: "failed to create trigger"}
	}
	def.AddTrigger(trigger)

	action, err := createAction(src, dest, overwrite)
	if err != nil {
		return taskmaster.RegisteredTask{}, &ErrCreateTaskFailure{Inner: err, Message: "failed to create action"}
	}
	def.AddAction(action)

	def.Principal.RunLevel = taskmaster.TASK_RUNLEVEL_HIGHEST
	// S4U is a necessary workaround to suppress powershell from flashing up when executing
	def.Principal.LogonType = taskmaster.TASK_LOGON_S4U
	def.Settings.AllowDemandStart = true
	def.Settings.AllowHardTerminate = false
	def.Settings.DontStartOnBatteries = false
	def.Settings.Enabled = true
	// Src and Dest together make a backup task unique
	def.Settings.MultipleInstances = taskmaster.TASK_INSTANCES_IGNORE_NEW
	def.Settings.StopIfGoingOnBatteries = false
	def.Settings.WakeToRun = false

	def.RegistrationInfo.Documentation = src + `|` + dest

	createdTask, _, err := conn.CreateTask(fPath+"\\"+parseTaskPath(src, dest), def, true)
	if err != nil {
		return taskmaster.RegisteredTask{}, &ErrCreateTaskFailure{Inner: err, Message: "failed to create task"}
	}
	return createdTask, nil
}

func DeleteScheduledTask(tName string) error {
	conn, err := taskmaster.Connect()
	if err != nil {
		return &ErrConnectSchedulerFailure{Inner: err, Message: "failed to connect to task scheduler"}
	}
	defer conn.Disconnect()

	err = conn.DeleteTask(fPath + "\\" + tName)
	if err != nil {
		return &ErrDeleteTaskFailure{Inner: err, Message: "Failed to delete task"}
	}
	return nil
}
