// Package main provides various examples of Fyne API capabilities.
package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/cmd/fyne_settings/settings"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/axgle/mahonia"
	"github.com/flopp/go-findfont"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type diagonal struct {
}

func (d *diagonal) MinSize(objects []fyne.CanvasObject) fyne.Size {
	w, h := float32(0), float32(0)
	for _, o := range objects {
		childSize := o.MinSize()
		w += childSize.Width
		h += childSize.Height
	}
	return fyne.NewSize(w, h)
}
func (d *diagonal) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	pos := fyne.NewPos(3, 3)
	objects[0].Resize(fyne.NewSize(600, 40))
	objects[1].Resize(fyne.NewSize(600, 480))
	objects[0].Move(pos)
	objects[1].Move(fyne.NewPos(3, 50))
}

type User struct {
	gorm.Model
	Name   string
	Status string
}

type Mission struct {
	gorm.Model
	Name     string
	UserName string
	Batch    string
	Status   string
}

type Menus struct {
	Title string
	View  func(w fyne.Window) fyne.CanvasObject
}

var (
	DB    *gorm.DB
	menus = map[string]Menus{
		"welcome": {"主页", welcomeScreen},
		"user":    {"人员列表", userScreen},
		"mission": {"任务列表", missionScreen},
	}
	menuIndex = map[string][]string{
		"": {"welcome", "user", "mission"},
	}
)

const preferenceCurrentTutorial = "currentTutorial"

func main() {

	fontPaths := findfont.List()
	for _, path := range fontPaths {
		if strings.Contains(path, "simkai.ttf") {
			os.Setenv("FYNE_FONT", path)
			break
		}
	}

	db, err := gorm.Open(sqlite.Open("mission.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	DB = db
	// Migrate the schema
	db.AutoMigrate(&User{}, &Mission{})
	// var users = []User{{Name: "WU", Status: "启用"}, {Name: "LI", Status: "启用"}, {Name: "WANG", Status: "启用"}}
	// db.Create(&users)

	a := app.NewWithID("lewis-lau")
	// a.SetIcon(theme.FyneLogo())
	w := a.NewWindow("任务分配程序")
	w.SetMainMenu(makeMenu(a, w))
	w.SetMaster()

	content := container.NewMax()
	title := widget.NewLabel("Component name")
	setTutorial := func(t Menus) {
		title.SetText(t.Title)

		content.Objects = []fyne.CanvasObject{t.View(w)}
		content.Refresh()
	}

	tutorial := container.NewBorder(
		container.NewVBox(title, widget.NewSeparator()), nil, nil, nil, content)
	split := container.NewHSplit(makeNav(setTutorial, true), tutorial)
	split.Offset = 0.2
	w.SetContent(split)
	w.Resize(fyne.NewSize(800, 600))
	w.ShowAndRun()
}

func makeMenu(a fyne.App, w fyne.Window) *fyne.MainMenu {
	settingsItem := fyne.NewMenuItem("Setting", func() {
		w := a.NewWindow("Fyne Settings")
		w.SetContent(settings.NewSettings().LoadAppearanceScreen(w))
		w.Resize(fyne.NewSize(480, 480))
		w.Show()
	})
	importItem := fyne.NewMenuItem("导入客户", func() {
		w := a.NewWindow("批量导入客户")
		file := widget.NewButton("请导入CSV文件", func() {
			file_Dialog := dialog.NewFileOpen(
				func(r fyne.URIReadCloser, _ error) {
					// read files
					csvReader := csv.NewReader(r)
					records, err := csvReader.ReadAll()
					if err != nil {
						log.Fatal("Unable to parse file as CSV for ", err)
					}
					assignMission(records)
					w.Close()
				}, w)
			file_Dialog.SetFilter(storage.NewExtensionFileFilter([]string{".csv"}))
			noUserDialog := dialog.NewInformation("没有用户", "当前没有启用的用户,请先增加用户", w)
			var user User
			DB.Where("status = '启用'").First(&user)
			if user.ID == 0 {
				noUserDialog.Show()
			} else {
				file_Dialog.Show()
			}
		})
		box := container.New(layout.NewCenterLayout(), file)
		w.SetContent(box)
		w.Resize(fyne.NewSize(550, 350))
		w.Show()
	})
	// a quit item will be appended to our first (File) menu
	file := fyne.NewMenu("File", settingsItem, importItem)
	return fyne.NewMainMenu(
		file,
	)
}

func makeNav(setTutorial func(tutorial Menus), loadPrevious bool) fyne.CanvasObject {
	a := fyne.CurrentApp()

	tree := &widget.Tree{
		ChildUIDs: func(uid string) []string {
			return menuIndex[uid]
		},
		IsBranch: func(uid string) bool {
			children, ok := menuIndex[uid]

			return ok && len(children) > 0
		},
		CreateNode: func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("Collection Widgets")
		},
		UpdateNode: func(uid string, branch bool, obj fyne.CanvasObject) {
			t, ok := menus[uid]
			if !ok {
				fyne.LogError("Missing tutorial panel: "+uid, nil)
				return
			}
			obj.(*widget.Label).SetText(t.Title)
		},
		OnSelected: func(uid string) {
			if t, ok := menus[uid]; ok {
				a.Preferences().SetString(preferenceCurrentTutorial, uid)
				setTutorial(t)
			}
		},
	}

	if loadPrevious {
		currentPref := a.Preferences().StringWithFallback(preferenceCurrentTutorial, "welcome")
		tree.Select(currentPref)
	}

	themes := container.New(layout.NewGridLayout(2),
		widget.NewButton("Dark", func() {
			a.Settings().SetTheme(theme.DarkTheme())
		}),
		widget.NewButton("Light", func() {
			a.Settings().SetTheme(theme.LightTheme())
		}),
	)

	return container.NewBorder(nil, themes, nil, nil, tree)
}

func welcomeScreen(_ fyne.Window) fyne.CanvasObject {

	return container.NewCenter(container.NewVBox(
		widget.NewLabelWithStyle("任务分配规则:平均随机分配任务到每个用户", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
	))
}

func userScreen(w fyne.Window) fyne.CanvasObject {
	users := []User{}
	DB.Find(&users)

	list := widget.NewList(
		func() int {
			return len(users)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(widget.NewIcon(theme.DocumentIcon()), widget.NewLabel("Template Object"))
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			item.(*fyne.Container).Objects[1].(*widget.Label).SetText(users[id].Name + "(" + users[id].Status + ")")
		},
	)
	selectedID := widget.NewEntry()
	selectedID.Hide()
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("姓名")
	options := []string{"启用", "禁用"}
	statusEntry := widget.NewSelect(options, nil)
	saveButton := widget.NewButton("新增", func() {
		if nameEntry.Text != "" && statusEntry.Selected != "" {
			if selectedID.Text == "" {
				var user User
				DB.Where("name = ?", nameEntry.Text).First(&user)
				if user.ID != 0 {
					noUserDialog := dialog.NewInformation("用户冲突", "已存在同名用户", w)
					noUserDialog.Show()
				} else {
					userInfo := User{Name: nameEntry.Text, Status: statusEntry.Selected}
					DB.Create(&userInfo)
				}
				DB.Find(&users)
			} else {
				userInfo := User{}
				DB.First(&userInfo, selectedID.Text)
				userInfo.Name = nameEntry.Text
				userInfo.Status = statusEntry.Selected
				DB.Save(&userInfo)
				DB.Find(&users)
			}
		}
	})
	clearButton := widget.NewButton("清除", func() {
		nameEntry.SetText("")
		statusEntry.SetSelected("启用")
		selectedID.SetText("")
		saveButton.SetText("新增")
	})
	vbox := container.NewVBox(nameEntry, statusEntry, selectedID, saveButton, clearButton)

	list.OnSelected = func(id widget.ListItemID) {
		nameEntry.SetText(users[id].Name)
		statusEntry.SetSelected(users[id].Status)
		selectedID.SetText(fmt.Sprintf("%v", users[id].ID))
		saveButton.SetText("修改")
	}
	list.OnUnselected = func(id widget.ListItemID) {
		nameEntry.SetText("")
		statusEntry.SetSelected("启用")
		selectedID.SetText("")
		saveButton.SetText("新增")
	}

	return container.NewHSplit(list, container.NewCenter(vbox))
}

func missionScreen(_ fyne.Window) fyne.CanvasObject {
	missions := []Mission{}
	DB.Find(&missions)
	t := widget.NewTable(
		func() (int, int) { return len(missions) + 1, 3 },
		func() fyne.CanvasObject {
			return widget.NewLabel("Cell 000, 000")
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*widget.Label)
			switch id.Col {
			case 0:
				if id.Row == 0 {
					label.SetText("批次号")
				} else {
					label.SetText(fmt.Sprintf("%v", missions[id.Row-1].Batch))
				}
			case 1:
				if id.Row == 0 {
					label.SetText("负责人")
				} else {
					label.SetText(fmt.Sprintf("%v", missions[id.Row-1].UserName))
				}
			case 2:
				if id.Row == 0 {
					label.SetText("客户名称")
				} else {
					label.SetText(fmt.Sprintf("%v", missions[id.Row-1].Name))
				}
			}
		},
	)

	batchEntry := widget.NewEntry()
	batchEntry.SetPlaceHolder("批次号")
	missionEntry := widget.NewEntry()
	missionEntry.SetPlaceHolder("客户名称")
	userEntry := widget.NewEntry()
	userEntry.SetPlaceHolder("负责人名称")
	searchButton := widget.NewButtonWithIcon("搜索", theme.SearchIcon(), func() {
		userWhere := Mission{}
		if batchEntry.Text != "" {
			userWhere.Batch = batchEntry.Text
		}
		if missionEntry.Text != "" {
			userWhere.Name = missionEntry.Text
		}
		if userEntry.Text != "" {
			userWhere.UserName = userEntry.Text
		}
		DB.Where(&userWhere).Find(&missions)
		t.Refresh()
	})
	clearButton := widget.NewButtonWithIcon("清除", theme.CancelIcon(), func() {
		batchEntry.SetText("")
		missionEntry.SetText("")
		userEntry.SetText("")
	})

	hbox := container.NewGridWithColumns(5, batchEntry, userEntry, missionEntry, searchButton, clearButton)
	return container.New(&diagonal{}, hbox, t)
}

func assignMission(records [][]string) {

	var missionArr []int
	var resArr [][]string
	decoder := mahonia.NewDecoder("GBK")

	for j := 1; j < len(records); j++ {
		missionArr = append(missionArr, j)
	}
	var header []string
	header = append(header, "批次号", "任务名", "负责人")
	resArr = append(resArr, header)
	shuffled := shuffle(missionArr)
	for i := 0; i < len(shuffled); i++ {
		userName := getNextUser()
		name := decoder.ConvertString(records[shuffled[i]][1])
		batch := records[shuffled[i]][0]
		mission := Mission{Batch: batch, Name: name, UserName: userName, Status: "ok"}
		fmt.Println(mission)
		DB.Create(&mission)
		var a []string
		a = append(a, batch, name, userName)
		resArr = append(resArr, a)
	}
	file, err := os.Create("分配结果.csv")
	if err != nil {
		log.Fatalln("failed to open file", err)
	}
	defer file.Close()
	file.WriteString("\xEF\xBB\xBF")
	w := csv.NewWriter(file)
	defer w.Flush()
	w.WriteAll(resArr)
}
func shuffle(src []int) []int {
	final := make([]int, len(src))
	rand.Seed(time.Now().UTC().UnixNano())
	perm := rand.Perm(len(src))

	for i, v := range perm {
		final[v] = src[i]
	}
	return final
}

func getNextUser() string {
	var lastUser Mission
	var name string
	DB.Where("status = 'ok'").Order("updated_at desc").First(&lastUser)
	lastName := lastUser.UserName
	if lastName == "" {
		var user User
		DB.Where("status = '启用'").First(&user)
		name = user.Name
	} else {
		var user User
		var newUser User
		DB.Where("name = ?", lastName).First(&user)
		lastID := &user.ID
		DB.Where("status = '启用' AND id > ?", lastID).First(&newUser)
		if newUser.ID == 0 {
			DB.Where("status = '启用'").First(&newUser)
		}
		name = newUser.Name
	}
	fmt.Println(name)
	return name
}
