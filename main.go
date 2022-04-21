package main

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	g "github.com/AllenDang/giu"
	"github.com/Coffee4Coffee/GoBackup/scheduler"
	"github.com/capnspacehook/taskmaster"
	"github.com/sqweek/dialog"
)

var (
	srcDir              string
	destDir             string
	weekdays            []string
	monthlyDays         []string
	backupLimitOptions  []string
	hours               []string
	scheduledTasks      taskmaster.RegisteredTaskCollection
	tableData           []*g.TableRowWidget
	overwrite           bool
	disabled            bool
	weekdaySelected     int32
	backupLimitSelected int32
	monthlyDaySelected  int32
	hourSelected        int32
	radioOp             int

	user32         = syscall.NewLazyDLL("user32.dll")
	procMessageBox = user32.NewProc("MessageBoxW")
)

// https://docs.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-messagebox
const (
	MB_RETRYCANCEL = 0x00000005
	MB_ICONERROR   = 0x00000010
	MB_DEFBUTTON2  = 0x00000100
	IDCANCEL       = 2
	IDRETRY        = 4
)

func MessageBox(caption, text string, flags uint) int {
	hwnd := uintptr(0)
	ret, _, _ := procMessageBox.Call(
		hwnd,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(caption))),
		uintptr(flags))

	return int(ret)
}

func handleError(err error) int {
	switch err.(type) {
	case *scheduler.ErrConnectSchedulerFailure:
		return MessageBox("Connection Error", "Could not connect to the windows task scheduler\nDo you want to try again?", MB_RETRYCANCEL|MB_ICONERROR|MB_DEFBUTTON2)
	case *scheduler.ErrRetrieveTaskFolderFailure, *scheduler.ErrRetrieveTasksFailure:
		return MessageBox("Fetch Error", "Could not fetch the scheduled backup tasks\nDo you want to try again?", MB_RETRYCANCEL|MB_ICONERROR|MB_DEFBUTTON2)
	case *scheduler.ErrCreateTaskFailure:
		return MessageBox("Create Error", "Could not create the scheduled backup task\nDo you want to try again?", MB_RETRYCANCEL|MB_ICONERROR|MB_DEFBUTTON2)
	case *scheduler.ErrDeleteTaskFailure:
		return MessageBox("Delete Error", "Could not delete the scheduled backup task\nDo you want to try again?", MB_RETRYCANCEL|MB_ICONERROR|MB_DEFBUTTON2)
	case *scheduler.ErrDeleteTaskFolderFailure:
		return MessageBox("Delete Error", "Could not delete the task folder\nDo you want to try again?", MB_RETRYCANCEL|MB_ICONERROR|MB_DEFBUTTON2)
	default:
		return MessageBox("Unknown Error", "An unknown error occurred\nPlease restart the application and try again", MB_ICONERROR)
	}
}

func selectFolder(src bool) {
	directory, _ := dialog.Directory().Title("Select the folder").Browse()
	if src {
		srcDir = directory
	} else {
		destDir = directory
	}
	checkReady()
}

func resetForm() {
	srcDir = ""
	destDir = ""
	monthlyDaySelected = 0
	weekdaySelected = 0
	overwrite = false
	hourSelected = 0
	radioOp = 0
	disabled = true
}

func initializeOptions() {
	// Create Task (Form ready)
	disabled = true

	backupLimitOptions = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "∞"}

	// taskmaster.LastDayOfTheMonth does not work currently, leave it out
	monthlyDays = make([]string, 31)
	for i := 0; i < 31; i++ {
		monthlyDays[i] = strconv.Itoa(i + 1)
	}

	// Weekdays
	weekdays = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

	// Hours
	for i := 0; i < 24; i++ {
		if i <= 9 {
			hours = append(hours, "0"+strconv.Itoa(i)+":00")
		} else {
			hours = append(hours, strconv.Itoa(i)+":00")
		}
	}
}

func getTriggerIntervalType(tr taskmaster.Trigger) string {
	switch tr.(type) {
	default:
		return "Unknown"
	case taskmaster.DailyTrigger:
		return "Daily"
	case taskmaster.WeeklyTrigger:
		return "Weekly"
	case taskmaster.MonthlyTrigger:
		return "Monthly"
	}
}

func updateTable() {
	if len(tableData) > 0 {
		tableData = tableData[:0]
	}
	for index, _task := range scheduledTasks {
		// Closure needed
		task := _task
		key := index

		doc := strings.Split(task.Definition.RegistrationInfo.Documentation, "|")
		srcPath := doc[0]
		destPath := doc[1]
		tableData = append(tableData, g.TableRow(
			g.Label(srcPath),
			g.Tooltip(srcPath),
			g.Label(destPath),
			g.Tooltip(destPath),
			g.Label(getTriggerIntervalType(task.Definition.Triggers[0])),
			g.Label(task.NextRunTime.Format("2006-01-02 15:04:05")),
			g.Label(task.LastRunTime.Format("2006-01-02 15:04:05")),
			g.Label(strconv.Itoa(int(task.MissedRuns))),
			g.Label(task.LastTaskResult.String()),
			g.Button("Delete").OnClick(func() { g.OpenPopup(strconv.Itoa(key)) }),
			g.PopupModal(strconv.Itoa(key)).Flags(g.WindowFlagsNoTitleBar|g.WindowFlagsNoResize|g.WindowFlagsNoMove).Layout(
				g.Label("Are you sure?"),
				g.Row(
					g.Button("Yes").Size(60, 30).OnClick(func() {
						deleteScheduledBackup(key)
						g.CloseCurrentPopup()
					}),
					g.Button("No").Size(60, 30).OnClick(func() { g.CloseCurrentPopup() }),
				),
			),
		))
	}
}

func initializeTable() {
	var err error
	scheduledTasks, err = scheduler.GetAllScheduledTasks()
	if err != nil {
		if messageBoxReturnCode := handleError(err); messageBoxReturnCode == IDCANCEL {
			os.Exit(1)
		} else if messageBoxReturnCode == IDRETRY {
			initializeTable()
		} else {
			os.Exit(1)
		}
	}
	updateTable()
}

func setDayOption() g.Layout {
	if radioOp == 1 {
		return g.Layout{
			g.Row(
				g.Label("Weekday"),
			),
			g.Row(
				g.Combo("", weekdays[weekdaySelected], weekdays, &weekdaySelected).Size(110),
			),
		}
	}
	if radioOp == 2 {
		return g.Layout{
			g.Row(
				g.Label("Day of the month"),
			),
			g.Row(
				g.Combo("", monthlyDays[monthlyDaySelected], monthlyDays, &monthlyDaySelected).Size(110),
				g.Tooltip("Months that do not include the given day will be skipped, e.g. the 30th for february"),
			),
		}
	}
	return g.Layout{}
}

func showLimitOption() g.Layout {
	if !overwrite {
		return g.Layout{
			g.Row(
				g.Label("limit"),
			),
			g.Row(
				g.Combo("", backupLimitOptions[backupLimitSelected], backupLimitOptions, &backupLimitSelected).Size(110),
				g.Tooltip("Set a limit to the number of backups. Older backups are removed upon exceeding this limit"),
			),
		}
	} else {
		return g.Layout{}
	}
}

func deleteScheduledBackup(index int) {
	deleteFolder := false
	if len(scheduledTasks) == 1 {
		deleteFolder = true
	}
	err := scheduler.DeleteScheduledTask(scheduledTasks[index].Name, deleteFolder)
	if err != nil {
		if messageBoxReturnCode := handleError(err); messageBoxReturnCode == IDRETRY {
			deleteScheduledBackup(index)
		} else if messageBoxReturnCode == IDCANCEL {
			return
		} else {
			os.Exit(1)
		}
	} else {
		initializeTable()
	}
}

func createScheduledBackup() {
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		MessageBox("Directory Error", "The given src directoy does not exist\nPlease restart the application and try again", MB_ICONERROR)
		os.Exit(1)
	}
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		MessageBox("Directory Error", "The given dest directoy does not exist\nPlease restart the application and try again", MB_ICONERROR)
		os.Exit(1)
	}

	_, err := scheduler.CreateScheduledTask(
		scheduler.TriggerType(radioOp),
		uint8(monthlyDaySelected),
		uint8(weekdaySelected),
		uint8(hourSelected),
		uint8(backupLimitSelected+1),
		srcDir,
		destDir,
		overwrite,
	)
	if err != nil {
		if messageBoxReturnCode := handleError(err); messageBoxReturnCode == IDRETRY {
			createScheduledBackup()
		} else if messageBoxReturnCode != IDCANCEL && messageBoxReturnCode != IDRETRY {
			os.Exit(1)
		}
	} else {
		initializeTable()
		resetForm()
	}
}

func checkReady() {
	if len(srcDir) > 0 && len(destDir) > 0 {
		disabled = false
	} else {
		disabled = true
	}
}

func loop() {
	g.SingleWindow().Layout(
		g.Row(
			g.Align(g.AlignCenter).To(
				g.Column(
					g.Row(
						g.Label("Select a folder to backup"),
					),
					g.Row(

						g.InputTextMultiline(&srcDir).Size(1000, 30).Flags(g.InputTextFlagsReadOnly),
						g.Button("Select").Size(100, 30).OnClick(func() { selectFolder(true) }),
					),
				),
				g.Dummy(0, 10),
				g.Column(
					g.Row(
						g.Label("Select a destination for the backup"),
					),
					g.Row(
						g.InputTextMultiline(&destDir).Size(1000, 30).Flags(g.InputTextFlagsReadOnly),
						g.Button("Select").Size(100, 30).OnClick(func() { selectFolder(false) }),
						g.Tooltip("The backup folder will be created if it does not already exist"),
					),
				),
				g.Dummy(0, 10),
				g.Column(
					g.Row(
						g.Label("Select an interval"),
					),
					g.Row(
						g.RadioButton("Monthly", radioOp == 2).OnChange(func() {
							radioOp = 2
							setDayOption()
						}),
						g.RadioButton("Weekly", radioOp == 1).OnChange(func() {
							radioOp = 1
							setDayOption()
						}),
						g.RadioButton("Daily", radioOp == 0).OnChange(func() {
							radioOp = 0
							setDayOption()
						}),
					),
				),
				g.Dummy(0, 10),
				g.Column(
					g.Row(
						setDayOption(),
						g.Label("Time"),
						g.Combo("", hours[hourSelected], hours, &hourSelected).Size(100),
						g.Checkbox("Overwrite", &overwrite),
						g.Tooltip("Overwrite the previous backup folder with a new one, or create a new backup folder with a timestamp on every execution"),
						showLimitOption(),
					),
				),
				g.Dummy(0, 30),
				g.Column(
					g.Row(
						g.Button("Create backup").Size(200, 50).OnClick(createScheduledBackup).Disabled(disabled),
					),
				),
			),
		),
		g.Dummy(0, 100),
		g.Row(
			g.Table().
				Columns(
					g.TableColumn("Src").Flags(g.TableColumnFlagsWidthStretch),
					g.TableColumn("Dest").Flags(g.TableColumnFlagsWidthFixed),
					g.TableColumn("Time interval").Flags(g.TableColumnFlagsWidthFixed),
					g.TableColumn("Next Run Time").Flags(g.TableColumnFlagsWidthFixed),
					g.TableColumn("Last Run Time").Flags(g.TableColumnFlagsWidthFixed),
					g.TableColumn("Missed Runs").Flags(g.TableColumnFlagsWidthFixed),
					g.TableColumn("Last Task Result").Flags(g.TableColumnFlagsWidthFixed),
					g.TableColumn("Delete").Flags(g.TableColumnFlagsWidthFixed),
				).
				Rows(
					tableData...,
				),
		),
	)
}

func main() {
	if runtime.GOOS != "windows" {
		dialog.Message(runtime.GOOS + " is currently not supported by this application").Title("OS not supported").Error()
		os.Exit(1)
	}
	initializeOptions()
	initializeTable()

	w := g.NewMasterWindow("GoBackup", 1600, 800, 0)
	w.Run(loop)
}
