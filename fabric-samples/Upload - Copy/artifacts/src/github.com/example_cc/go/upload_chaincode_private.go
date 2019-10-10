/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// ====CHAINCODE EXECUTION SAMPLES (CLI) ==================

// ==== Invoke Upload, pass private data as base64 encoded bytes in transient map ====
//
// export Upload=$(echo -n "{\"name\":\"Upload1\",\"color\":\"blue\",\"size\":35,\"owner\":\"tom\",\"price\":99}" | base64)
// peer chaincode invoke -C mychannel -n Uploadp -c '{"Args":["initUpload"]}' --transient "{\"Upload\":\"$Upload\"}"
//
// export Upload=$(echo -n "{\"name\":\"Upload2\",\"color\":\"red\",\"size\":50,\"owner\":\"tom\",\"price\":102}" | base64)
// peer chaincode invoke -C mychannel -n Uploadp -c '{"Args":["initUpload"]}' --transient "{\"Upload\":\"$Upload\"}"
//
// export Upload=$(echo -n "{\"name\":\"Upload3\",\"color\":\"blue\",\"size\":70,\"owner\":\"tom\",\"price\":103}" | base64)
// peer chaincode invoke -C mychannel -n Uploadp -c '{"Args":["initUpload"]}' --transient "{\"Upload\":\"$Upload\"}"
//
// export Upload_OWNER=$(echo -n "{\"name\":\"Upload2\",\"owner\":\"jerry\"}" | base64)
// peer chaincode invoke -C mychannel -n Uploadp -c '{"Args":["transferUpload"]}' --transient "{\"Upload_owner\":\"$Upload_OWNER\"}"
//
// export Upload_DELETE=$(echo -n "{\"name\":\"Upload1\"}" | base64)
// peer chaincode invoke -C mychannel -n Uploadp -c '{"Args":["delete"]}' --transient "{\"Upload_delete\":\"$Upload_DELETE\"}"

// ==== Query Upload, since queries are not recorded on chain we don't need to hide private data in transient map ====
// peer chaincode query -C mychannel -n Uploadp -c '{"Args":["readUpload","Upload1"]}'
// peer chaincode query -C mychannel -n Uploadp -c '{"Args":["readUploadPrivateDetails","Upload1"]}'
// peer chaincode query -C mychannel -n Uploadp -c '{"Args":["getUploadByRange","Upload1","Upload4"]}'
//
// Rich Query (Only supported if CouchDB is used as state database):
//   peer chaincode query -C mychannel -n Uploadp -c '{"Args":["queryUploadByOwner","tom"]}'
//   peer chaincode query -C mychannel -n Uploadp -c '{"Args":["queryUpload","{\"selector\":{\"owner\":\"tom\"}}"]}'

// INDEXES TO SUPPORT COUCHDB RICH QUERIES
//
// Indexes in CouchDB are required in order to make JSON queries efficient and are required for
// any JSON query with a sort. As of Hyperledger Fabric 1.1, indexes may be packaged alongside
// chaincode in a META-INF/statedb/couchdb/indexes directory. Or for indexes on private data
// collections, in a META-INF/statedb/couchdb/collections/<collection_name>/indexes directory.
// Each index must be defined in its own text file with extension *.json with the index
// definition formatted in JSON following the CouchDB index JSON syntax as documented at:
// http://docs.couchdb.org/en/2.1.1/api/database/find.html#db-index
//
// This Upload02_private example chaincode demonstrates a packaged index which you
// can find in META-INF/statedb/couchdb/collection/collectionUpload/indexes/indexOwner.json.
// For deployment of chaincode to production environments, it is recommended
// to define any indexes alongside chaincode so that the chaincode and supporting indexes
// are deployed automatically as a unit, once the chaincode has been installed on a peer and
// instantiated on a channel. See Hyperledger Fabric documentation for more details.
//
// If you have access to the your peer's CouchDB state database in a development environment,
// you may want to iteratively test various indexes in support of your chaincode queries.  You
// can use the CouchDB Fauxton interface or a command line curl utility to create and update
// indexes. Then once you finalize an index, include the index definition alongside your
// chaincode in the META-INF/statedb/couchdb/indexes directory or
// META-INF/statedb/couchdb/collections/<collection_name>/indexes directory, for packaging
// and deployment to managed environments.
//
// In the examples below you can find index definitions that support Upload02_private
// chaincode queries, along with the syntax that you can use in development environments
// to create the indexes in the CouchDB Fauxton interface.
//

//Example hostname:port configurations to access CouchDB.
//
//To access CouchDB docker container from within another docker container or from vagrant environments:
// http://couchdb:5984/
//
//Inside couchdb docker container
// http://127.0.0.1:5984/

// Index for docType, owner.
// Note that docType and owner fields must be prefixed with the "data" wrapper
//
// Index definition for use with Fauxton interface
// {"index":{"fields":["data.docType","data.owner"]},"ddoc":"indexOwnerDoc", "name":"indexOwner","type":"json"}

// Index for docType, owner, size (descending order).
// Note that docType, owner and size fields must be prefixed with the "data" wrapper
//
// Index definition for use with Fauxton interface
// {"index":{"fields":[{"data.size":"desc"},{"data.docType":"desc"},{"data.owner":"desc"}]},"ddoc":"indexSizeSortDoc", "name":"indexSizeSortDesc","type":"json"}

// Rich Query with index design doc and index name specified (Only supported if CouchDB is used as state database):
//   peer chaincode query -C mychannel -n Uploadp -c '{"Args":["queryUpload","{\"selector\":{\"docType\":\"Upload\",\"owner\":\"tom\"}, \"use_index\":[\"_design/indexOwnerDoc\", \"indexOwner\"]}"]}'

// Rich Query with index design doc specified only (Only supported if CouchDB is used as state database):
//   peer chaincode query -C mychannel -n Uploadp -c '{"Args":["queryUpload","{\"selector\":{\"docType\":{\"$eq\":\"Upload\"},\"owner\":{\"$eq\":\"tom\"},\"size\":{\"$gt\":0}},\"fields\":[\"docType\",\"owner\",\"size\"],\"sort\":[{\"size\":\"desc\"}],\"use_index\":\"_design/indexSizeSortDoc\"}"]}'

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

type Upload struct {
	ObjectType string `json:"docType"` //docType is used to distinguish the various types of objects in state database
	Name       string `json:"name"`    //the fieldtags are needed to keep case from bouncing around
	Owner      string `json:"owner"`
}

type UploadPrivateDetails struct {
	ObjectType string `json:"docType"` //docType is used to distinguish the various types of objects in state database
	Name       string `json:"name"`    //the fieldtags are needed to keep case from bouncing around
	Hash      string `json:"hash"`
}

// ===================================================================================
// Main
// ===================================================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// Init initializes chaincode
// ===========================
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
/*	logger.Info("########### example_cc0 Init ###########")
	_, args := stub.GetFunctionAndParameters()
	var configName ,configPath string
	
	//Initialize chaincode
	configName = args[0]
	configPath = args[1]
	err = stub.PutState(configName, configPath)
	if err != nil {
		return shim.Error(err.Error())
	}*/
	return shim.Success(nil)
}

// Invoke - Our entry point for Invocations
// ========================================
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println("invoke is running " + function)

	// Handle different functions
	switch function {
	case "initUpload":
		//create a new Upload
		return t.initUpload(stub, args)
	case "readUpload":
		//read a Upload
		return t.readUpload(stub, args)
	case "readUploadPrivateDetails":
		//read a Upload private details
		return t.readUploadPrivateDetails(stub, args)
	case "transferUpload":
		//change owner of a specific Upload
		return t.transferUpload(stub, args)
	case "delete":
		//delete a Upload
		return t.delete(stub, args)
	case "queryUploadByOwner":
		//find Upload for owner X using rich query
		return t.queryUploadByOwner(stub, args)
	case "queryUpload":
		//find Upload based on an ad hoc rich query
		return t.queryUpload(stub, args)
	case "getUploadByRange":
		//get Upload based on range query
		return t.getUploadByRange(stub, args)
	default:
		//error
		fmt.Println("invoke did not find func: " + function)
		return shim.Error("Received unknown function invocation")
	}
}

// ============================================================
// initUpload - create a new Upload, store into chaincode state
// ============================================================
func (t *SimpleChaincode) initUpload(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

	type UploadTransientInput struct {
		Name  string `json:"name"` //the fieldtags are needed to keep case from bouncing around
		Hash  string `json:"hash"`
		Owner string `json:"owner"`
				
	}

	// ==== Input sanitation ====
	fmt.Println("- start init Upload")

	if len(args) != 0 {
		return shim.Error("Incorrect number of arguments. Private Upload data must be passed in transient map.")
	}

	transMap, err := stub.GetTransient()
	if err != nil {
		return shim.Error("Error getting transient: " + err.Error())
	}

//	if(transMap == nil)
	//	fmt.Println("transMap is nil ")
	/*
	var buffer = new Buffer(transMap.map.conversation.value.toArrayBuffer());
// from buffer into string
var JSONString = buffer.toString(‘utf8’);
// from json string into object
var JSONObject = JSON.parse(JSONString);
*/
	if _, ok := transMap["Upload"]; !ok {
		return shim.Error("Upload must be a key in the transient map")
	//return shim.Error(transMap["Upload"]);
	}

	if len(transMap["Upload"]) == 0 {
		return shim.Error("Upload value in the transient map must be a non-empty JSON string")
	}

	var UploadInput UploadTransientInput
	err = json.Unmarshal(transMap["Upload"], &UploadInput)
	if err != nil {
		return shim.Error("Failed to decode JSON of: " + string(transMap["Upload"]))
	}

	fmt.Println("Values : Name: " + UploadInput.Name+" Hash:"+UploadInput.Hash +" Owner:"+UploadInput.Owner)
	UploadInput.Name = "Pan"
	UploadInput.Hash ="This is a hash code"
	UploadInput.Owner="garima"
	if len(UploadInput.Name) == 0 {
		return shim.Error("name field must be a non-empty string")
	}
	if len(UploadInput.Hash) == 0 {
		return shim.Error("owner field must be a non-empty string")
	}
	if len(UploadInput.Owner) == 0 {
		return shim.Error("owner field must be a non-empty string")
	}


	// ==== Check if Upload already exists ====
	UploadAsBytes, err := stub.GetPrivateData("collectionUpload", UploadInput.Name)
	if err != nil {
		return shim.Error("Failed to get Upload: " + err.Error())
	} else if UploadAsBytes != nil {
		fmt.Println("This Upload already exists: " + UploadInput.Name)
		return shim.Error("This Upload already exists: " + UploadInput.Name)
	}

	// ==== Create Upload object, marshal to JSON, and save to state ====
	Upload := &Upload{
		ObjectType: "Upload",
		Name:       UploadInput.Name,
		Owner:      UploadInput.Owner,
	}
	UploadJSONasBytes, err := json.Marshal(Upload)
	if err != nil {
		return shim.Error(err.Error())
	}

	// === Save Upload to state ===
	err = stub.PutPrivateData("collectionUpload", UploadInput.Name, UploadJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	// ==== Create Upload private details object with price, marshal to JSON, and save to state ====
	UploadPrivateDetails := &UploadPrivateDetails{
		ObjectType: "UploadPrivateDetails",
		Name:       UploadInput.Name,
		Hash:      UploadInput.Hash,
	}
	UploadPrivateDetailsBytes, err := json.Marshal(UploadPrivateDetails)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutPrivateData("collectionUploadPrivateDetails", UploadInput.Name, UploadPrivateDetailsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	//  ==== Index the Upload to enable color-based range queries, e.g. return all blue Upload ====
	//  An 'index' is a normal key/value entry in state.
	//  The key is a composite key, with the elements that you want to range query on listed first.
	//  In our case, the composite key is based on indexName~color~name.
	//  This will enable very efficient state range queries based on composite keys matching indexName~color~*
	/*
	indexName := "color~name"
	colorNameIndexKey, err := stub.CreateCompositeKey(indexName, []string{Upload.Color, Upload.Name})
	if err != nil {
		return shim.Error(err.Error())
	}
	//  Save index entry to state. Only the key name is needed, no need to store a duplicate copy of the Upload.
	//  Note - passing a 'nil' value will effectively delete the key from state, therefore we pass null character as value
	value := []byte{0x00}
	stub.PutPrivateData("collectionUpload", colorNameIndexKey, value)
	*/
	// ==== Upload saved and indexed. Return success ====
	fmt.Println("- end init Upload")
	return shim.Success(nil)
}

// ===============================================
// readUpload - read a Upload from chaincode state
// ===============================================
func (t *SimpleChaincode) readUpload(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the Upload to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetPrivateData("collectionUpload", name) //get the Upload from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + name + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Upload does not exist: " + name + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(valAsbytes)
}

// ===============================================
// readUploadreadUploadPrivateDetails - read a Upload private details from chaincode state
// ===============================================
func (t *SimpleChaincode) readUploadPrivateDetails(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the Upload to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetPrivateData("collectionUploadPrivateDetails", name) //get the Upload private details from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get private details for " + name + ": " + err.Error() + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Upload private details does not exist: " + name + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(valAsbytes)
}

// ==================================================
// delete - remove a Upload key/value pair from state
// ==================================================
func (t *SimpleChaincode) delete(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	fmt.Println("- start delete Upload")

	type UploadDeleteTransientInput struct {
		Name string `json:"name"`
	}

	if len(args) != 0 {
		return shim.Error("Incorrect number of arguments. Private Upload name must be passed in transient map.")
	}

	transMap, err := stub.GetTransient()
	if err != nil {
		return shim.Error("Error getting transient: " + err.Error())
	}

	if _, ok := transMap["Upload_delete"]; !ok {
		return shim.Error("Upload_delete must be a key in the transient map")
	}

	if len(transMap["Upload_delete"]) == 0 {
		return shim.Error("Upload_delete value in the transient map must be a non-empty JSON string")
	}

	var UploadDeleteInput UploadDeleteTransientInput
	err = json.Unmarshal(transMap["Upload_delete"], &UploadDeleteInput)
	if err != nil {
		return shim.Error("Failed to decode JSON of: " + string(transMap["Upload_delete"]))
	}

	if len(UploadDeleteInput.Name) == 0 {
		return shim.Error("name field must be a non-empty string")
	}

	// to maintain the color~name index, we need to read the Upload first and get its color
	valAsbytes, err := stub.GetPrivateData("collectionUpload", UploadDeleteInput.Name) //get the Upload from chaincode state
	if err != nil {
		return shim.Error("Failed to get state for " + UploadDeleteInput.Name)
	} else if valAsbytes == nil {
		return shim.Error("Upload does not exist: " + UploadDeleteInput.Name)
	}

	var UploadToDelete Upload
	err = json.Unmarshal([]byte(valAsbytes), &UploadToDelete)
	if err != nil {
		return shim.Error("Failed to decode JSON of: " + string(valAsbytes))
	}

	// delete the Upload from state
	err = stub.DelPrivateData("collectionUpload", UploadDeleteInput.Name)
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}

	// Also delete the Upload from the color~name index
	//indexName := "color~name"
	//colorNameIndexKey, err := stub.CreateCompositeKey(indexName, []string{UploadToDelete.Color, UploadToDelete.Name})
	/*if err != nil {
		return shim.Error(err.Error())
	}*/
	/*err = stub.DelPrivateData("collectionUpload", colorNameIndexKey)
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}
*/
	// Finally, delete private details of Upload
	err = stub.DelPrivateData("collectionUploadPrivateDetails", UploadDeleteInput.Name)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// ===========================================================
// transfer a Upload by setting a new owner name on the Upload
// ===========================================================
func (t *SimpleChaincode) transferUpload(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	fmt.Println("- start transfer Upload")

	type UploadTransferTransientInput struct {
		Name  string `json:"name"`
		Owner string `json:"owner"`
	}

	if len(args) != 0 {
		return shim.Error("Incorrect number of arguments. Private Upload data must be passed in transient map.")
	}

	transMap, err := stub.GetTransient()
	if err != nil {
		return shim.Error("Error getting transient: " + err.Error())
	}

	if _, ok := transMap["Upload_owner"]; !ok {
		return shim.Error("Upload_owner must be a key in the transient map")
	}

	if len(transMap["Upload_owner"]) == 0 {
		return shim.Error("Upload_owner value in the transient map must be a non-empty JSON string")
	}

	var UploadTransferInput UploadTransferTransientInput
	err = json.Unmarshal(transMap["Upload_owner"], &UploadTransferInput)
	if err != nil {
		return shim.Error("Failed to decode JSON of: " + string(transMap["Upload_owner"]))
	}

	if len(UploadTransferInput.Name) == 0 {
		return shim.Error("name field must be a non-empty string")
	}
	if len(UploadTransferInput.Owner) == 0 {
		return shim.Error("owner field must be a non-empty string")
	}

	UploadAsBytes, err := stub.GetPrivateData("collectionUpload", UploadTransferInput.Name)
	if err != nil {
		return shim.Error("Failed to get Upload:" + err.Error())
	} else if UploadAsBytes == nil {
		return shim.Error("Upload does not exist: " + UploadTransferInput.Name)
	}

	UploadToTransfer := Upload{}
	err = json.Unmarshal(UploadAsBytes, &UploadToTransfer) //unmarshal it aka JSON.parse()
	if err != nil {
		return shim.Error(err.Error())
	}
	UploadToTransfer.Owner = UploadTransferInput.Owner //change the owner

	UploadJSONasBytes, _ := json.Marshal(UploadToTransfer)
	err = stub.PutPrivateData("collectionUpload", UploadToTransfer.Name, UploadJSONasBytes) //rewrite the Upload
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println("- end transferUpload (success)")
	return shim.Success(nil)
}

// ===========================================================================================
// getUploadByRange performs a range query based on the start and end keys provided.

// Read-only function results are not typically submitted to ordering. If the read-only
// results are submitted to ordering, or if the query is used in an update transaction
// and submitted to ordering, then the committing peers will re-execute to guarantee that
// result sets are stable between endorsement time and commit time. The transaction is
// invalidated by the committing peers if the result set has changed between endorsement
// time and commit time.
// Therefore, range queries are a safe option for performing update transactions based on query results.
// ===========================================================================================
func (t *SimpleChaincode) getUploadByRange(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	startKey := args[0]
	endKey := args[1]

	resultsIterator, err := stub.GetPrivateDataByRange("collectionUpload", startKey, endKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	fmt.Printf("- getUploadByRange queryResult:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

// =======Rich queries =========================================================================
// Two examples of rich queries are provided below (parameterized query and ad hoc query).
// Rich queries pass a query string to the state database.
// Rich queries are only supported by state database implementations
//  that support rich query (e.g. CouchDB).
// The query string is in the syntax of the underlying state database.
// With rich queries there is no guarantee that the result set hasn't changed between
//  endorsement time and commit time, aka 'phantom reads'.
// Therefore, rich queries should not be used in update transactions, unless the
// application handles the possibility of result set changes between endorsement and commit time.
// Rich queries can be used for point-in-time queries against a peer.
// ============================================================================================

// ===== Example: Parameterized rich query =================================================
// queryUploadByOwner queries for Upload based on a passed in owner.
// This is an example of a parameterized query where the query logic is baked into the chaincode,
// and accepting a single query parameter (owner).
// Only available on state databases that support rich query (e.g. CouchDB)
// =========================================================================================
func (t *SimpleChaincode) queryUploadByOwner(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0
	// "bob"
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	owner := strings.ToLower(args[0])

	queryString := fmt.Sprintf("{\"selector\":{\"docType\":\"Upload\",\"owner\":\"%s\"}}", owner)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

// ===== Example: Ad hoc rich query ========================================================
// queryUpload uses a query string to perform a query for Upload.
// Query string matching state database syntax is passed in and executed as is.
// Supports ad hoc queries that can be defined at runtime by the client.
// If this is not desired, follow the queryUploadForOwner example for parameterized queries.
// Only available on state databases that support rich query (e.g. CouchDB)
// =========================================================================================
func (t *SimpleChaincode) queryUpload(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0
	// "queryString"
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	queryString := args[0]

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

// =========================================================================================
// getQueryResultForQueryString executes the passed in query string.
// Result set is built and returned as a byte array containing the JSON results.
// =========================================================================================
func getQueryResultForQueryString(stub shim.ChaincodeStubInterface, queryString string) ([]byte, error) {

	fmt.Printf("- getQueryResultForQueryString queryString:\n%s\n", queryString)

	resultsIterator, err := stub.GetPrivateDataQueryResult("collectionUpload", queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryRecords
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	fmt.Printf("- getQueryResultForQueryString queryResult:\n%s\n", buffer.String())

	return buffer.Bytes(), nil
}
