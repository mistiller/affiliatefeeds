package awinclient

import (
	"fmt"
	"net/url"
	"os"
	"testing"

	"stillgrove.com/gofeedyourself/pkg/feedservice/feed"
)

func TestAwinClient(t *testing.T) {
	var (
		err      error
		c        *Client
		products []Product
	)

	loc, err := feed.NewLocale("SE", "sv", "sv_se")
	if err != nil {
		t.Fatal(err)
	}
	c, err = New(
		loc,
		os.Getenv("AWIN_TOKEN"),
		os.Getenv("AWIN_FEED_TOKEN"),
	)
	if err != nil {
		t.Fatal(err)
	}

	c.AddApiRequest(
		"GET",
		"publishers/668643/programmes",
		&url.Values{
			"relationship": []string{"joined"},
		},
		nil,
	)
	_, err = c.Execute(false)
	if err != nil {
		t.Fatal(err)
	}

	/*progs, err = c.GetProgrammes()
	if err != nil {
		t.Fatal(err)
	}

	// IMPORTANT: If there are no programmes to be found, just end the test
	if len(progs) == 0 {
		log.Println("No programmes returned")
		return
	}

	fmt.Println("Joined Programmes:")
	for i := range progs {
		fmt.Println(progs[i].ProgrammeInfo.Name)
	}*/

	fmt.Println("Found feeds:")
	fds, err := c.GetFeeds()
	for i := range fds {
		fmt.Println(fds[i].AdvertiserName)
	}

	products, err = c.GetProducts(500)
	if err != nil {
		t.Fatal(err)
	}

	for i := range products {
		if i == 0 {
			fmt.Println(products[i])
		}
		err = products[i].Validate()
		if err != nil {
			t.Fatal(err)
		}
	}

	/*b, err := json.Marshal(products)
	checkErr(err, t)

	path := helpers.FindFolderDir("gofeedyourself") + "/logs/awin_test.json"
	f, err := os.Create(path)
	checkErr(err, t)

	_, err = f.Write(b)
	checkErr(err, t)*/
}

/*
func TestAwinReport(t *testing.T) {
	c := New(
		os.Getenv("AWIN_TOKEN"),
		os.Getenv("AWIN_FEED_TOKEN"),
	)

	trx, err := c.GetTransactions()
	checkErr(err, t)
	fmt.Println(trx)
}
*/

//https://productdata.awin.com/datafeed/download/apikey/8f796c80b23a8b5811fe01f69e75f1d1/language/any/fid/20775/columns/aw_deep_link,product_name,aw_product_id,merchant_product_id,merchant_image_url,description,merchant_category,search_price,merchant_name,merchant_id,category_name,category_id,aw_image_url,currency,store_price,delivery_cost,merchant_deep_link,language,last_updated,display_price,data_feed_id,colour,brand_name,brand_id,keywords,product_type,promotional_text,model_number,commission_group,rrp_price,saving,savings_percent,base_price,base_price_amount,base_price_text,product_price_old,in_stock,stock_quantity,valid_from,valid_to,is_for_sale,web_offer,pre_order,stock_status,size_stock_status,size_stock_amount,merchant_thumb_url,large_image,alternate_image,aw_thumb_url,alternate_image_two,alternate_image_three,alternate_image_four,reviews,average_rating,rating,number_available,custom_1,custom_2,custom_3,custom_4,custom_5,custom_6,custom_7,custom_8,custom_9,ean,isbn,upc,mpn,parent_product_id,product_GTIN,Fashion%3Asuitable_for,Fashion%3Acategory,Fashion%3Asize,Fashion%3Amaterial,Fashion%3Apattern,Fashion%3Aswatch,delivery_restrictions,delivery_weight,warranty,terms_of_contract,delivery_time,merchant_product_category_path,merchant_product_second_category,merchant_product_third_category,product_short_description,specifications,condition,product_model,dimensions,basket_link/format/csv/delimiter/%2C/compression/gzip/adultcontent/1/
