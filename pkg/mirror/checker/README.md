# CDN检测原理

## 检测工具
- API站点[https://www.17ce.com](https://www.17ce.com)
- [开发文档](https://www.17ce.com/soft/17CE_WS_API_v2.12.pdf)

## 详细说明
- 获取目标站点源变更列表
  ```go
  func getChangeList() ([]string, error) {
	// 访问目标站点并获取HTML页面文件
  	resp, err := http.Get(srcURL)
  	...
	// 解析HTML文件并获取a标签下的href属性，再从中获取JSON文件的URL
  	doc, err := goquery.NewDocumentFromReader(resp.Body)
  	...
  	var result []string
  	doc.Find("a").Each(func(i int, selection *goquery.Selection) {
  		href, ok := selection.Attr("href")
  		...
  		if strings.HasSuffix(href, ".json") && href != "current.json" {
  			result = append(result, href)
  		}
  	})
  	return result, nil
  }
  ```
- 获取源变更列表中每一项的变更文件列表
  ```go
  func getChangeFiles() ([]string, error) {
  	...
  	var changeMetaInfoList []ChangeMetaInfo
  	for _, name := range changeList {
  		...
  	}
  	...
	// 获取源最近更新
  	var recentlyChanges []string
  	for i := len(changeMetaInfoList) - 1; i >= 0; i-- {
  		...
  	}
  	...
  }
  ```
- 获取目标站点的CDN的DNS的IP列表
  ```go
  func prefetchCdnDns(host string) error {
	ips, ok := dnsCache[host]
  	...
	// 利用检测工具检测DNS，开发文档见以上说明
  	ips, err = checker.CheckTool.TestDNS(host)
    ...
  }
  ```
- 检测镜像的同步进度并将进度信息更新至数据库
  ```go
  func testAllMirrors(mirrors0 mirrors, validateInfoList []*FileValidateInfo) {
  	...
	// 用于记录同步所用时间
  	t0 := time.Now()
	// 检测结果列表
  	var testResults []*TestResult
	// 同步锁用于限制对检测结果列表的并发写入
  	var mu sync.Mutex
  
  	for _, mirror := range mirrors0 {
  		...
		// 将检测镜像的任务放入任务队列并发执行以减少同步任务时间
  		pool.JobQueue <- func() {
  			...
			// 对镜像列表项逐一检测
  			testResult := testMirror(mirrorCopy.Id, mirrorCopy.GetUrlPrefix(),
  				mirrorCopy.Weight, validateInfoList)
  			testMirrorFinish()
  			duration0 := time.Since(t0)
  			...
			// 写入检测结果
  			mu.Lock()
  			testResults = append(testResults, testResult...)
  			mu.Unlock()
  			pool.JobDone()
  		}
  	}
  	pool.WaitAll()
    // 持久化更新后的镜像同步进度
  	checker.pushAllMirrorsTestResults(testResults)
  }
  ```