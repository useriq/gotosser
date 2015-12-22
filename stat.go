package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hhkbp2/go-strftime"
)

type dirStatInfo struct {
	//количество файлов
	Count int64
	//дата последней передачи файла
	LastProcessingDate int64
	//общий размер файлов
	TotalSize int64
}

type TosserStat struct {
	Dates      map[string]map[string]*dirStatInfo
	ConfigName string
}

func (ts *TosserStat) update(file string, size int64) {
	dir := filepath.Dir(file)
	now := strftime.Format("%Y-%m-%d", time.Now())

	if ts.Dates[now] == nil {
		ts.Dates[now] = make(map[string]*dirStatInfo)
	}

	if ts.Dates[now][dir] == nil {
		ts.Dates[now][dir] = new(dirStatInfo)
	}

	ts.Dates[now][dir].Count++
	ts.Dates[now][dir].LastProcessingDate = time.Now().Unix()
	ts.Dates[now][dir].TotalSize += size
}

func (ts *TosserStat) load(ConfigName string) (err error) {
	var file []byte
	if file, err = ioutil.ReadFile(ConfigName); err != nil {
		return err
	}
	x := new(TosserStat)
	if err = json.Unmarshal(file, &x); err != nil {
		return err
	}
	ts.Dates = x.Dates
	return nil
}

func (ts *TosserStat) save(ConfigName string) {
	b, err := json.Marshal(ts)
	if err != nil {
		log.Println("Не удалось сохранить статистику в файл. ", err)
	}
	err = ioutil.WriteFile(ConfigName, b, 0666)
	if err != nil {
		log.Println("Не удалось сохранить статистику в файл. ", err)
	}
}

//NewTosserStat - создает, инициализирует и возвращает ссылку на TosserStat
func NewTosserStat(ConfigName string) *TosserStat {
	if _, err := os.Stat(filepath.Dir(ConfigName)); err != nil {
		err := os.MkdirAll(filepath.Dir(ConfigName), os.ModeDir)
		if err != nil {
			log.Println(err)
		}
	}
	ts := new(TosserStat)
	err := ts.load(ConfigName)
	if err != nil {
		ts.Dates = make(map[string]map[string]*dirStatInfo)
	}
	ts.ConfigName = ConfigName
	return ts
}

// SaveStatLoop - запуск горутины в которой периодически будет сохраняться файл
func SaveStatLoop(tosserstat *TosserStat) chan processingItem {
	savestatchan := make(chan processingItem)
	go func(tosserstat *TosserStat) {
		//каждые 10 секунд записываем статистику на диск, если есть, что писать
		t := time.Tick(10 * time.Second)
		i := 0
		for {
			select {
			case <-t:
				if i != 0 {
					tosserstat.save(tosserstat.ConfigName)
					i = 0
				}
			case item := <-savestatchan:
				tosserstat.update(item.fullSrcFilePath, item.size)
				i++
			}
		}
	}(tosserstat)
	return savestatchan
}
