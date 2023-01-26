package requestbuilder

const bidRequestTemplateText3 = ("{" +
	`"openrtb":{` + (`"ver":"3.0",` +
	`"domainspec":"adcom",` +
	`"domainver":"1.0",` +
	`"request":{` + (`"id":"<RequestID>",` +
	`"tmax":51,` +
	`"at":1,` +
	`"cur":[` + (`"USD"`) +
	`],` +
	`"source":{` + (`"tid":"1lmo0hJYZX5eH3BPmqHSVzYfSGa",` +
	`"ts":<TimeStamp>,` +
	`"pchain":"1lmo0cdhb6woJTWl0Bouj5dXR5b"`) +
	`},` +
	`"item":[` + (`{` + (`"id":"<ItemID>",` +
	`"qty":2,` +
	`"flr":3.58,` +
	`"flrcur":"USD",` +
	`"exp":4,` +
	`"spec":{` + (`"placement":{` + (`"tagid":"<TagID>",` +
	`"ssai":1,` +
	`"display":{` + (`"mime":"text/html",` +
	`"w":1767,` +
	`"h":1092,` +
	`"unit":1`) +
	`}`) +
	`}`) +
	`}`) +
	`}`) +
	`],` +
	`"context":{` + (`"site":{` + (`"domain":"example.com",` +
	`"page":"http://easy.example.com/easy?cu=13824;cre=mu;target=_blank",` +
	`"ref":"http://tpc.googlesyndication.com/pagead/js/loader12.html?http://sdk.streamrail.com/vpaid/js/668/sam.js",` +
	`"cat":[` + (`"1",` +
	`"33",` +
	`"544",` +
	`"765",` +
	`"1222",` +
	`"1124",` +
	`"789",` +
	`"995",` +
	`"133",` +
	`"45",` +
	`"76",` +
	`"91"` +
	`],`) +
	`"pub":{` + (`"id":"qqwer1234xgfd",` +
	`"name":"site_name",` +
	`"domain":"my.site.com"` +
	`}`) +
	`},`) +
	`"user":{` + (`"id":"<UserID>",` +
	`"buyeruid":"<BuyerID>",` +
	`"yob":1961,` +
	`"gender":"F",` +
	`"data":[` + (`{` + (`"id":"pub-demographics",` +
	`"name":"data_name",` +
	`"segment":[` + (`{` + (`"id":"345qw245wfrtgwertrt56765wert",` +
	`"name":"segment_name",` +
	`"value":"segment_value"` +
	`}`) +
	`]`) +
	`}`) +
	`]`) +
	`},`) +
	`"device":{` + (`"type":2,` +
	`"ifa":"<DeviceIFA>",` +
	`"os":2,` +
	`"lang":"en",` +
	`"osv":"10",` +
	`"model":"browser",` +
	`"make":"desktop",` +
	`"ip":"104.4.9.67",` +
	`"ua":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36",` +
	`"geo":{` + (`"lat":37.789,` +
	`"lon":-122.394,` +
	`"country":"USA",` +
	`"city":"San Francisco",` +
	`"region":"CA",` +
	`"zip":"94105",` +
	`"type":2,` +
	`"ext":{` + (`"dma":650,` +
	`"state":"oklahoma",` +
	`"continent":"north america"` +
	`}`) +
	`}`) +
	`},`) +
	`"restrictions":{` + (`"cattax":2,` +
	`"bcat":[` + (`"12",` +
	`"143",` +
	`"34",` +
	`"887",` +
	`"122",` +
	`"999",` +
	`"1023",` +
	`"13",` +
	`"4",` +
	`"565",` +
	`"920",` +
	`"224",` +
	`"857",` +
	`"1320"` +
	`],`) +
	`"badv":[` + (`"facebook.com",` +
	`"twitter.com",` +
	`"google.com",` +
	`"amazon.com",` +
	`"youtube.com"` +
	`]`) +
	`},`) +
	`"regs":{` + (`"coppa":0,` +
	`"gdpr":0,` +
	`"ext":{` + (`"sb568":0` +
	`}`) +
	`}`) +
	`}`) +
	`}`) +
	`}`) +
	`}`)

const bidRequestTemplateText2 = ("{" +
	`"at":1,` +
	`"badv":[` +
	`"facebook.com",` +
	`"twitter.com",` +
	`"google.com",` +
	`"amazon.com",` +
	`"youtube.com"` +
	`],` +
	`"bcat":[` +
	`"12",` +
	`"143",` +
	`"34",` +
	`"887",` +
	`"122",` +
	`"999",` +
	`"1023",` +
	`"13",` +
	`"4",` +
	`"565",` +
	`"920",` +
	`"224",` +
	`"857",` +
	`"1320"` +
	`],` +
	`"cur":[` +
	`"USD"` +
	`],` +
	`"device":{` +
	`"devicetype":2,` +
	`"ifa":"<DeviceIFA>",` +
	`"ip":"104.4.9.67",` +
	`"language":"en",` +
	`"make":"desktop",` +
	`"model":"browser",` +
	`"os":"iOS",` +
	`"osv":"10",` +
	`"ua":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36"` +
	`},` +
	`"ext":{` +
	`"cattax":2` +
	`},` +
	`"id":"<RequestID>",` +
	`"imp":[` +
	`{` +
	`"banner":{` +
	`"ext":{` +
	`"qty":2,` +
	`"unit":1` +
	`},` +
	`"h":1092,` +
	`"mimes":"text/html",` +
	`"w":1767` +
	`},` +
	`"bidfloor":3.58,` +
	`"bidfloorcur":"USD",` +
	`"exp":4,` +
	`"ext":{` +
	`"ssai":1` +
	`},` +
	`"id":"<ItemID>",` +
	`"tagid":"<TagID>"` +
	`}` +
	`],` +
	`"regs":{` +
	`"coppa":0,` +
	`"ext":{` +
	`"gdpr":0,` +
	`"sb568":0` +
	`}` +
	`},` +
	`"site":{` +
	`"cat":[` +
	`"1",` +
	`"33",` +
	`"544",` +
	`"765",` +
	`"1222",` +
	`"1124",` +
	`"789",` +
	`"995",` +
	`"133",` +
	`"45",` +
	`"76",` +
	`"91"` +
	`],` +
	`"domain":"example.com",` +
	`"page":"http://easy.example.com/easy?cu=13824;cre=mu;target=_blank",` +
	`"publisher":{` +
	`"domain":"my.site.com",` +
	`"id":"qqwer1234xgfd",` +
	`"name":"site_name"` +
	`},` +
	`"ref":"http://tpc.googlesyndication.com/pagead/js/loader12.html?http://sdk.streamrail.com/vpaid/js/668/sam.js"` +
	`},` +
	`"source":{` +
	`"ext":{` +
	`"ts":<TimeStamp>` +
	`},` +
	`"pchain":"1lmo0cdhb6woJTWl0Bouj5dXR5b",` +
	`"tid":"1lmo0hJYZX5eH3BPmqHSVzYfSGa"` +
	`},` +
	`"tmax":51,` +
	`"user":{` +
	`"buyeruid":"<BuyerID>",` +
	`"data":[` +
	`{` +
	`"id":"pub-demographics",` +
	`"name":"data_name",` +
	`"segment":[` +
	`{` +
	`"id":"345qw245wfrtgwertrt56765wert",` +
	`"name":"segment_name",` +
	`"value":"segment_value"` +
	`}` +
	`]` +
	`}` +
	`],` +
	`"gender":"F",` +
	`"geo":{` +
	`"city":"San Francisco",` +
	`"country":"USA",` +
	`"ext":{` +
	`"continent":"north america",` +
	`"dma":650,` +
	`"state":"oklahoma"` +
	`},` +
	`"lat":37.789,` +
	`"lon":-122.394,` +
	`"region":"CA",` +
	`"type":2,` +
	`"zip":"94105"` +
	`},` +
	`"id":"<UserID>",` +
	`"yob":1961` +
	`}` +
	"}")
