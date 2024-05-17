package goweb_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/protamail/goweb/htm"
)

type Result = htm.Result

var el, attr, add = htm.NewElem, htm.NewAttr, htm.Append
var printf, itoa = fmt.Sprintf, strconv.Itoa
var text, uric, id = htm.Text, htm.EncodeURIComponent, htm.AsIs

func Test1(t *testing.T) {
	type B struct {
		a string
		B int
	}
	var b = B{"heh", 2}
	fmt.Printf(htm.See(1, b))

	//var r Result
	//
	//		<html class="heh" data-href="sdsd?sds=1">
	//			<body>
	//				<nav class="heh" data-href="sdsd?sds=1">
	//					<div>
	//						<ul>
	//							<li data-href="hj&'gjh&ha=wdfw eee"></li>
	//							<img src="j">
	//							<br>
	//							<span data-href="ddd">dsdsdsd</span>
	//							...
	//						</ul>
	//					</div>
	//				</nav>
	//			</body>
	//		</html>
	//
	a := make([]string, 1000, 1000)
	for i := 0; i < 1000; i++ {
		r :=
			el("html", attr("class=", "heh", "data-href=", "sdsd?sds=1"),
				el("body", "",
					el("nav", attr("class=", "heh", "data-href=", "sdsd?sds=1"),
						el("div", "",
							/*el("ul", "", func() Result {
								var result = htm.NewHTML(1000)
								for j := 0; j < 1000; j++ {
									result = add(result,
										el("li", attr("data-href=", uric(`hj&"'>gjh`)+`&ha=`+uric(`wdfw&`)+func() string {
											if true {
												return "&eee"
											}
											return ""
										}()),
											text(printf("%d", j)),
											el("img", attr("src=", printf("img%d", j))),
											el("img", attr("src=", itoa(j))),
											el(`img`, attr("src=", printf("img%.2f", float32(j)))),
											el("br", ""),
											el("div", "", text("heh"), id("da"), text("boom")),
											el("span", attr("data-href", "ddd"), text("dsdsi&dsd")),
										),
									)
								}
								return result
							}()),*/
							el("ul", "", htm.Map(a, func(j int) Result {
								return el("li", attr("data-href=", uric(`hj&"'>gjh`)+`&ha=`+uric(`wdfw&`)+func() string {
									if true {
										return "&eee"
									}
									return ""
								}()),
									text(printf("%d", j)),
									el("img", attr("src=", printf("img%d", j))),
									el("img", attr("src=", itoa(j))),
									el(`img`, attr("src=", printf("img%.2f", float32(j)))),
									el("br", ""),
									el("div", "", text("heh"), id("da"), text("boom")),
									el("span", attr("data-href", "ddd"), text("dsdsi&dsd"), text(a[j])),
								)
							})),
						),
					),
				),
			)
		_ = r
		//		fmt.Println(r.String())
	}
}

func aTest2(t *testing.T) {
	var buckets = []map[string]string{
		{"bucket": "WLGCRU", "bucketName": "Wireline Growth & CRU"},
		{"bucket": "TOTAL", "bucketName": "Total"},
	}
	var listHeader = func() Result {
		result :=
			el("tr", attr("class=", "tr-hdr trb-t trb-s trb-b narrow-font"),
				el("td", attr("class=", "tdb-l"), el("br", "")),
				el("td", "", id("PID")),
				el("td", "", id("RVP")),
				el("td", "", id("Sales Center")),
				func() Result {
					var result Result
					for _, b := range buckets {
						result = add(result, el("td", "", text(b["bucketName"])))
					}
					return result
				}())
		return result
	}
	fmt.Println(listHeader().String())
}
