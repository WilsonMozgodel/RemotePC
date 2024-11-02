package main

import (
	"bytes"
	"fmt"
	"image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kbinani/screenshot"
)

var wait = false
var waitreb = false

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic:", r)
		}
	}()

	addToStartup()

	bot, err := tgbotapi.NewBotAPI("")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			var msg tgbotapi.MessageConfig

			switch update.Message.Command() {
			case "start":
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Remote Control for PC")
				keyboard := tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Выключить компьютер"),
						tgbotapi.NewKeyboardButton("Перезагрузить компьютер"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Сделать скриншот"),
					),
				)
				msg.ReplyMarkup = keyboard

			default:
				if update.Message.Text == "Выключить компьютер" {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Вы точно хотите выключить ПК? Напишите 'Да' для подтверждения.")
					wait = true
					waitreb = false
				} else if update.Message.Text == "Да" {
					if wait {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Выключение системы...")
						bot.Send(msg)
						shutdownComputer()
					}
				}

				if update.Message.Text == "Перезагрузить компьютер" {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Вы точно хотите перезагрузить ПК? Напишите 'Да' для подтверждения.")
					waitreb = true
					wait = false
				} else if update.Message.Text == "Да" {
					if waitreb {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Перезагрузка системы...")
						bot.Send(msg)
						rebootComputer()
					}
				}

				if update.Message.Text == "Сделать скриншот" {
					imgBytes, err := takeScreenshot()
					if err != nil {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Не удалось сделать скриншот.")
					} else {
						photo := tgbotapi.NewPhoto(update.Message.Chat.ID, tgbotapi.FileBytes{Name: "screenshot.png", Bytes: imgBytes})
						if _, err := bot.Send(photo); err != nil {
							log.Println(err)
						}
						continue
					}
				}
			}

			if _, err := bot.Send(msg); err != nil {
				log.Println(err)
			}
		}
	}
}

func addToStartup() {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Ошибка получения пути к исполняемому файлу: %v", err)
	}

	startupPath := `C:\ProgramData\Microsoft\Windows\Start Menu\Programs\Startup`
	shortCutName := "RemotePC.lnk"

	shortCutPath := filepath.Join(startupPath, shortCutName)

	if _, err := os.Stat(shortCutPath); os.IsNotExist(err) {
		err = createShortcut(exePath, shortCutPath)
		if err != nil {
			log.Fatalf("Ошибка создания ярлыка: %v", err)
		}
		log.Println("Приложение добавлено в автозапуск.")
	}
}

func createShortcut(targetPath, shortcutPath string) error {
	psCommand := fmt.Sprintf(`
$WshShell = New-Object -ComObject WScript.Shell;
$Shortcut = $WshShell.CreateShortcut("%s");
$Shortcut.TargetPath = "%s";
$Shortcut.WorkingDirectory = "%s"; # Optional: Set working directory
$Shortcut.Save();
`, shortcutPath, targetPath, filepath.Dir(targetPath))

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-command", psCommand)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ошибка выполнения команды PowerShell: %v, вывод: %s", err, output)
	}
	return nil
}

func shutdownComputer() {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("shutdown", "/s", "/t", "0")
	case "linux":
		cmd = exec.Command("shutdown", "-h", "now")
	case "darwin":
		cmd = exec.Command("shutdown", "-h", "now")
	default:
		log.Println("Unsupported OS")
		return
	}

	if err := cmd.Run(); err != nil {
		log.Println("Error shutting down:", err)
	}
}

func rebootComputer() {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("shutdown", "/r", "/t", "0")
	case "linux":
		cmd = exec.Command("reboot")
	case "darwin":
		cmd = exec.Command("shutdown", "-r", "now")
	default:
		return
	}

	if err := cmd.Run(); err != nil {
		log.Println("Error rebooting:", err)
	}
}

func takeScreenshot() ([]byte, error) {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return nil, fmt.Errorf("no active displays found")
	}

	img, err := screenshot.CaptureDisplay(0)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
