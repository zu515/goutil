package rsyncutil

import (
	"container/list"
	"strings"
	"sync"

	belogs "github.com/astaxie/beego/logs"
)

type RsyncUrl struct {
	Url  string `json:"url"`
	Dest string `jsong:"dest"`
}

// queue for rsync url
type RsyncUrlQueue struct {
	Mutex    *sync.RWMutex
	WaitUrls *list.List
	CurUrls  *list.List
	UsedUrls *list.List

	Msg chan string // will trigger rsync
}

func NewQueue() *RsyncUrlQueue {
	m := new(sync.RWMutex)
	msg := make(chan string, 100)
	return &RsyncUrlQueue{
		WaitUrls: list.New(),
		CurUrls:  list.New(),
		UsedUrls: list.New(),
		Mutex:    m,
		Msg:      msg}
}
func (r *RsyncUrlQueue) WaitUrlsSize() int {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()
	return r.WaitUrls.Len()
}
func (r *RsyncUrlQueue) CurUrlsSize() int {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()
	return r.CurUrls.Len()
}
func (r *RsyncUrlQueue) UsedUrlsSize() int {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()
	return r.UsedUrls.Len()
}
func (r *RsyncUrlQueue) AddNewUrl(url string, dest string) *list.Element {
	belogs.Debug("AddNewUrl():url", url, "    dest:", dest)
	if len(url) == 0 || len(dest) == 0 {
		return nil
	}
	belogs.Debug("AddNewUrl():before lock ", url)
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	belogs.Debug("AddNewUrl():get WaitUrls check ", url)
	e := r.WaitUrls.Front()
	for e != nil {
		if strings.Contains(e.Value.(RsyncUrl).Url, url) {
			return nil
		} else {
			e = e.Next()
		}
	}
	belogs.Debug("AddNewUrl():get CurUrls check ", url)
	e = r.CurUrls.Front()
	for e != nil {
		if strings.Contains(e.Value.(RsyncUrl).Url, url) {
			return nil
		} else {
			e = e.Next()
		}
	}
	belogs.Debug("AddNewUrl():get UsedUrls check ", url)
	e = r.UsedUrls.Front()
	for e != nil {
		if strings.Contains(e.Value.(RsyncUrl).Url, url) {
			return nil
		} else {
			e = e.Next()
		}
	}
	rsync := RsyncUrl{Url: url, Dest: dest}
	belogs.Debug("AddNewUrl():add ", url)
	e = r.WaitUrls.PushBack(rsync)
	r.Msg <- "add"
	return e
}

func (r *RsyncUrlQueue) GetNextWaitUrls() []RsyncUrl {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	urls := make([]RsyncUrl, 0)
	var next *list.Element
	for e := r.WaitUrls.Front(); e != nil; e = next {
		next = e.Next()
		urls = append(urls, e.Value.(RsyncUrl))
		r.CurUrls.PushBack(e.Value.(RsyncUrl))
		r.WaitUrls.Remove(e)
	}

	return urls
}

func (r *RsyncUrlQueue) CurUrlsRsyncEnd(rsyncUrl RsyncUrl) {

	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	var next *list.Element
	for e := r.CurUrls.Front(); e != nil; e = next {
		next = e.Next()
		rsyncUrlCur := e.Value.(RsyncUrl)
		belogs.Debug("CurUrlsRsyncEnd():rsyncUrlCur", rsyncUrlCur, "    rsyncUrl:", rsyncUrl)

		if rsyncUrlCur.Url == rsyncUrl.Url {
			r.UsedUrls.PushBack(e.Value.(RsyncUrl))
			r.CurUrls.Remove(e)
			break
		}
	}
}

func (r *RsyncUrlQueue) GetWaitUrls() []RsyncUrl {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()

	urls := make([]RsyncUrl, 0)
	e := r.WaitUrls.Front()
	for e != nil {
		urls = append(urls, e.Value.(RsyncUrl))
		e = e.Next()
	}
	return urls
}
func (r *RsyncUrlQueue) GetCurUrls() []RsyncUrl {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()

	urls := make([]RsyncUrl, 0)
	e := r.CurUrls.Front()
	for e != nil {
		urls = append(urls, e.Value.(RsyncUrl))
		e = e.Next()
	}
	return urls
}
func (r *RsyncUrlQueue) GetUsedUrls() []RsyncUrl {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()

	urls := make([]RsyncUrl, 0)
	e := r.UsedUrls.Front()
	for e != nil {
		urls = append(urls, e.Value.(RsyncUrl))
		e = e.Next()
	}
	return urls
}