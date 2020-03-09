package dynamoconnection

import (
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	session "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"

	feed "stillgrove.com/gofeedyourself/pkg/feedservice/feed"
)

// DynamoConnection is the DynamoDB object which allows you to interact with the staging product table
type DynamoConnection struct {
	productMap        map[uint64]*feed.Product
	tableName         string //"test_products"
	ActiveSession     *session.Session
	queryLookbackDays int64
	credentials       *credentials.Credentials
	initialized       bool
}

// InitDynamoConnection creates a DynamoConnection object and return it
func InitDynamoConnection(id, secret, tableName string) (DynamoConnection, error) {
	var db DynamoConnection

	db.credentials = credentials.NewStaticCredentials(id, secret, "")
	db.tableName = tableName

	db.initialized = true

	return db, nil
}

// Get implements the feed interface and returns a product map;
// uses QueryLookbackDays and defaults it to 7
func (db DynamoConnection) Get() (map[uint64]*feed.Product, error) {
	if db.queryLookbackDays == 0 {
		db.queryLookbackDays = 2
	}
	if db.productMap == nil {
		err := db.getProducts(db.queryLookbackDays)
		if err != nil {
			return db.productMap, err
		}
	}
	return db.productMap, nil
}

// SetLookbackWindow sets the days to look back when querying for products (default = 2)
func (db *DynamoConnection) SetLookbackWindow(nDays int64) {
	db.queryLookbackDays = nDays
}

// GetTableDescription for basic diagnostics
func (db *DynamoConnection) GetTableDescription() error {
	if db.ActiveSession == nil {
		err := db.GetSession()
		if err != nil {
			return err
		}
	}
	svc := dynamodb.New(db.ActiveSession)

	req := &dynamodb.DescribeTableInput{
		TableName: aws.String(db.tableName),
	}
	result, err := svc.DescribeTable(req)
	if err != nil {
		return err
	}
	fmt.Print(result.Table)

	return nil
}

// UploadProducts takes a pointer to a product map and uploads it directly to dynamodb
func (db *DynamoConnection) UploadProducts(productMap map[uint64]*feed.Product) error {
	if db.ActiveSession == nil {
		err := db.GetSession()
		if err != nil {
			return err
		}
	}
	svc := dynamodb.New(db.ActiveSession)

	//l := len(productMap)
	for _, product := range productMap {
		av, err := dynamodbattribute.MarshalMap(product)

		if err != nil {
			fmt.Printf("Got error marshalling map -%v", err)
			continue
		}

		input := &dynamodb.PutItemInput{
			Item:      av,
			TableName: aws.String(db.tableName),
		}

		_, err = svc.PutItem(input)

		if err != nil {
			return err
		}

		//fmt.Println("Successfully added/updated", product.Name)
	}

	return nil
}

// Private Functions

func (db *DynamoConnection) GetSession() error {
	if db.initialized == false {
		return errors.New("Connection not initialized")
	}
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("eu-central-1"),
		Credentials: db.credentials,
		//Credentials: credentials.NewSharedCredentials("", db.SharedCredentialName),
	})
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	db.ActiveSession = sess

	return nil
}

// getProducts scans the whole table and returns a result filtered by a lookback window of n days
func (db *DynamoConnection) getProducts(queryLookbackDays int64) error {
	if db.ActiveSession == nil {
		err := db.GetSession()
		if err != nil {
			return err
		}
	}
	svc := dynamodb.New(db.ActiveSession)

	var ts int64
	ts = time.Now().Unix() - (86400 * queryLookbackDays)

	filt := expression.Name("lastSeen").GreaterThan(expression.Value(ts))

	expr, err := expression.NewBuilder().WithFilter(filt).Build()
	if err != nil {
		return err
	}

	params := &dynamodb.ScanInput{
		TableName:              aws.String(db.tableName),
		ReturnConsumedCapacity: aws.String("TOTAL"),
		ConsistentRead:         aws.Bool(true),
		//Limit: aws.Int64(50),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		//Select:	aws.String("COUNT"),
	}

	result, err := svc.Scan(params)
	if err != nil {
		return err
	}

	var obj = []feed.Product{}
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &obj)
	if err != nil {
		return err
	}

	db.productMap = map[uint64]*feed.Product{}
	var k uint64
	for idx := range obj {
		k = obj[idx].Key
		db.productMap[k] = &obj[idx]
	}
	return nil
}
