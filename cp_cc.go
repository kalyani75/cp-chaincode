/*
Copyright 2016 IBM

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Licensed Materials - Property of IBM
© Copyright IBM Corp. 2016
*/
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
	"math/rand"
	"time"

	"github.com/openblockchain/obc-peer/openchain/chaincode/shim"
)

var cqPrefix = "cq:"
var accountPrefix = "acct:"
var accountsKey = "accounts"

var recentLeapYear = 2016

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

func generateCUSIPSuffix(issueDate string, days int) (string, error) {

	t, err := msToTime(issueDate)
	if err != nil {
		return "", err
	}

	maturityDate := t.AddDate(0, 0, days)
	month := int(maturityDate.Month())
	day := maturityDate.Day()

	suffix := seventhDigit[month] + eigthDigit[day]
	return suffix, nil

}
func randomString(l int) string {
    bytes := make([]byte, l)
    for i := 0; i < l; i++ {
        bytes[i] = byte(randInt(65, 90))
    }
    return string(bytes)
}
func randInt(min int, max int) int {
    return min + rand.Intn(max-min)
}
const (
	millisPerSecond     = int64(time.Second / time.Millisecond)
	nanosPerMillisecond = int64(time.Millisecond / time.Nanosecond)
)

func msToTime(ms string) (time.Time, error) {
	msInt, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(msInt/millisPerSecond,
		(msInt%millisPerSecond)*nanosPerMillisecond), nil
}



type Owner struct {
	Company string    `json:"company"`
	//Quantity int      `json:"quantity"`
}

type Cheque struct {
	CUSIP       string   `json:"cusip"`
	Cheque_num string  `json:"ticker"`
	Par        float64 `json:"par"`	
	Owners    []Owner `json:"owner"`
	Issuer    string  `json:"issuer"`
	IssueDate string  `json:"issueDate"`
}

type Account struct {
	ID          string  `json:"id"`
	Prefix      string  `json:"prefix"`
	CashBalance float64 `json:"cashBalance"`
	AssetsIds   []string `json:"assetIds"`
}

type Transaction struct {
	CUSIP       string   `json:"cusip"`
	FromCompany string   `json:"fromCompany"`
	ToCompany   string   `json:"toCompany"`
	Quantity    int      `json:"quantity"`	
}

func (t *SimpleChaincode) createAccounts(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

	//  				0
	// "number of accounts to create"
	var err error
	numAccounts, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println("error creating accounts with input")
		return nil, errors.New("createAccounts accepts a single integer argument")
	}
	//create a bunch of accounts
	var account Account
	counter := 1
	for counter <= numAccounts {
		var prefix string
		suffix := "000A"
		if counter < 10 {
			prefix = strconv.Itoa(counter) + "0" + suffix
		} else {
			prefix = strconv.Itoa(counter) + suffix
		}
		var assetIds []string
		account = Account{ID: "company" + strconv.Itoa(counter), Prefix: prefix, CashBalance: 10000000.0, AssetsIds: assetIds}
		accountBytes, err := json.Marshal(&account)
		if err != nil {
			fmt.Println("error creating account" + account.ID)
			return nil, errors.New("Error creating account " + account.ID)
		}
		err = stub.PutState(accountPrefix+account.ID, accountBytes)
		counter++
		fmt.Println("created account" + accountPrefix + account.ID)
	}
	var blank []string
	blankBytes, _ := json.Marshal(&blank)
	err = stub.PutState("PaperKeys", blankBytes)

	fmt.Println("Accounts created")
	return nil, nil

}

func (t *SimpleChaincode) issueCheque(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

	/*		0
		json
	  	{
			"ticker":  "string",
			"par": 0.00,
			"qty": 10,
			"discount": 7.5,
			"maturity": 30,
			"owners": [ // This one is not required
				{
					"company": "company1",
					"quantity": 5
				},
				{
					"company": "company3",
					"quantity": 3
				},
				{
					"company": "company4",
					"quantity": 2
				}
			],				
			"issuer":"company2",
			"issueDate":"1456161763790"  (current time in milliseconds as a string)

		}
	*/
	//need one arg
	if len(args) != 1 {
		fmt.Println("error invalid arguments")
		return nil, errors.New("Incorrect number of arguments. Expecting cheque record")
	}

	var cq Cheque
	var err error
	var account Account

	fmt.Println("Unmarshalling Cheque");
	err = json.Unmarshal([]byte(args[0]), &cq)
	if err != nil {
		fmt.Println("error invalid paper issue")
		return nil, errors.New("Invalid commercial paper issue")
	}

	//generate the CUSIP
	//get account prefix
	fmt.Println("Getting state of - " + accountPrefix + cq.Issuer);
	accountBytes, err := stub.GetState(accountPrefix + cq.Issuer);
	if err != nil {
		fmt.Println("Error Getting state of - " + accountPrefix + cq.Issuer);
		return nil, errors.New("Error retrieving account " + cq.Issuer)
	}
	err = json.Unmarshal(accountBytes, &account)
	if err != nil {
		fmt.Println("Error Unmarshalling accountBytes");
		return nil, errors.New("Error retrieving account " + cq.Issuer)
	}
	fmt.Println("KD After Get State" + account)
	account.AssetsIds = append(account.AssetsIds, cq.CUSIP)

	// Set the issuer to be the owner of all quantity
	var owner Owner;
	owner.Company = cq.Issuer
	//owner.Quantity = cq.Qty
	
	cq.Owners = append(cq.Owners, owner)


	//commented by KD
	/*suffix, err := generateCUSIPSuffix(cq.IssueDate, cq.Maturity)
	if err != nil {
		fmt.Println("Error generating cusip");
		return nil, errors.New("Error generating CUSIP")
	}*/

	fmt.Println("Marshalling cq bytes");
	//cq.CUSIP = account.Prefix + suffix
	rand.Seed(time.Now().UTC().UnixNano())
	cq.CUSIP = randomString(10)

	fmt.Println("Getting State on cq " + cq.CUSIP)
	cqRxBytes, err := stub.GetState(cqPrefix+cq.CUSIP);
	if cqRxBytes == nil {
		fmt.Println("CUSIP does not exist, creating it")
		cqBytes, err := json.Marshal(&cq)
		if err != nil {
			fmt.Println("Error marshalling cq");
			return nil, errors.New("Error issuing Cheque")
		}
		err = stub.PutState(cqPrefix+cq.CUSIP, cqBytes)
		if err != nil {
			fmt.Println("Error issuing paper");
			return nil, errors.New("Error issuing Cheque")
		}

		fmt.Println("Marshalling account bytes to write");
		accountBytesToWrite, err := json.Marshal(&account)
		if err != nil {
			fmt.Println("Error marshalling account");
			return nil, errors.New("Error issuing Cheque")
		}
		err = stub.PutState(accountPrefix + cq.Issuer, accountBytesToWrite)
		if err != nil {
			fmt.Println("Error putting state on accountBytesToWrite");
			return nil, errors.New("Error issuing Cheque")
		}
		
		
		// Update the paper keys by adding the new key
		fmt.Println("Getting Cheque Keys");
		keysBytes, err := stub.GetState("PaperKeys")
		if err != nil {
			fmt.Println("Error retrieving paper keys");
			return nil, errors.New("Error retrieving Cheque keys")
		}
		var keys []string
		err = json.Unmarshal(keysBytes, &keys)
		if err != nil {
			fmt.Println("Error unmarshel keys");
			return nil, errors.New("Error unmarshalling Cheque keys ")
		}
		
		fmt.Println("Appending the new key to Cheque Keys");
		foundKey := false
		for _, key := range keys {
			if key == cqPrefix+cq.CUSIP {
				foundKey = true
			}
		}
		if foundKey == false {
			keys = append(keys, cqPrefix+cq.CUSIP);		
			keysBytesToWrite, err := json.Marshal(&keys)
			if err != nil {
				fmt.Println("Error marshalling keys");
				return nil, errors.New("Error marshalling the keys")
			}
			fmt.Println("Put state on PaperKeys");
			err = stub.PutState("PaperKeys", keysBytesToWrite)
			if err != nil {
				fmt.Println("Error writting keys back");
				return nil, errors.New("Error writing the keys back")
			}
		}
		
		fmt.Println("Issue commercial paper %+v\n", cq)
		return nil, nil
	} else {
		fmt.Println("CUSIP exists")
		
		var cqrx Cheque
		fmt.Println("Unmarshalling Cheque " + cq.CUSIP)
		err = json.Unmarshal(cpRxBytes, &cqrx)
		if err != nil {
			fmt.Println("Error unmarshalling cq " + cq.CUSIP)
			return nil, errors.New("Error unmarshalling cq " + cq.CUSIP)
		}
		//commented by KD
		/*cqrx.Qty = cqrx.Qty + cq.Qty;
		
		for key, val := range cqrx.Owners {
			if val.Company == cq.Issuer {
				cqrx.Owners[key].Quantity += cq.Qty
				break
			}
		}*/
				
		cpWriteBytes, err := json.Marshal(&cqrx)
		if err != nil {
			fmt.Println("Error marshalling cq");
			return nil, errors.New("Error issuing commercial paper")
		}
		err = stub.PutState(cqPrefix+cq.CUSIP, cpWriteBytes)
		if err != nil {
			fmt.Println("Error issuing paper");
			return nil, errors.New("Error issuing commercial paper")
		}

		fmt.Println("Updated commercial paper %+v\n", cqrx)
		return nil, nil
	}
}


func GetAllCheques(stub *shim.ChaincodeStub) ([]Cheque, error){
	
	var allCPs []Cheque
	
	// Get list of all the keys
	keysBytes, err := stub.GetState("PaperKeys")
	if err != nil {
		fmt.Println("Error retrieving paper keys")
		return nil, errors.New("Error retrieving paper keys")
	}
	var keys []string
	err = json.Unmarshal(keysBytes, &keys)
	if err != nil {
		fmt.Println("Error unmarshalling paper keys")
		return nil, errors.New("Error unmarshalling paper keys")
	}

	// Get all the cps
	for _, value := range keys {
		cpBytes, err := stub.GetState(value);
		
		var cp Cheque
		err = json.Unmarshal(cpBytes, &cp)
		if err != nil {
			fmt.Println("Error retrieving cp " + value)
			return nil, errors.New("Error retrieving cp " + value)
		}
		
		fmt.Println("Appending Cheque" + value)
		allCPs = append(allCPs, cp)
	}	
	
	return allCPs, nil
}

func GetCheque(cpid string, stub *shim.ChaincodeStub) (Cheque, error){
	var cp Cheque

	cpBytes, err := stub.GetState(cpid);
	if err != nil {
		fmt.Println("Error retrieving cp " + cpid)
		return cp, errors.New("Error retrieving cp " + cpid)
	}
		
	err = json.Unmarshal(cpBytes, &cp)
	if err != nil {
		fmt.Println("Error unmarshalling cp " + cpid)
		return cp, errors.New("Error unmarshalling cp " + cpid)
	}
		
	return cp, nil
}


func GetCompany(companyID string, stub *shim.ChaincodeStub) (Account, error){
	var company Account
	companyBytes, err := stub.GetState(accountPrefix+companyID);
	if err != nil {
		fmt.Println("Account not found " + companyID)
		return company, errors.New("Account not found " + companyID)
	}

	err = json.Unmarshal(companyBytes, &company)
	if err != nil {
		fmt.Println("Error unmarshalling account " + companyID)
		return company, errors.New("Error unmarshalling account " + companyID)
	}
	
	return company, nil
}


// Still working on this one
func (t *SimpleChaincode) encashCheque(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	/*		0
		json
	  	{
			  "CUSIP": "",
			  "fromCompany":"",
			  "toCompany":"",
			  "quantity": 1
		}
	*/
	//need one arg
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting cheque record")
	}
	
	var tr Transaction

	fmt.Println("Unmarshalling Transaction");
	err := json.Unmarshal([]byte(args[0]), &tr)
	if err != nil {
		fmt.Println("Error Unmarshalling Transaction");
		return nil, errors.New("Invalid cheque issue")
	}

	fmt.Println("Getting State on Cheque " + tr.CUSIP)
	cpBytes, err := stub.GetState(cqPrefix+tr.CUSIP);
	if err != nil {
		fmt.Println("CUSIP not found")
		return nil, errors.New("CUSIP not found " + tr.CUSIP)
	}

	var cp Cheque
	fmt.Println("Unmarshalling Cheque " + tr.CUSIP)
	err = json.Unmarshal(cpBytes, &cp)
	if err != nil {
		fmt.Println("Error unmarshalling cheque " + tr.CUSIP)
		return nil, errors.New("Error unmarshalling cheque " + tr.CUSIP)
	}

	var fromCompany Account
	fmt.Println("Getting State on fromCompany " + tr.FromCompany)	
	fromCompanyBytes, err := stub.GetState(accountPrefix+tr.FromCompany);
	if err != nil {
		fmt.Println("Account not found " + tr.FromCompany)
		return nil, errors.New("Account not found " + tr.FromCompany)
	}

	fmt.Println("Unmarshalling FromCompany ")
	err = json.Unmarshal(fromCompanyBytes, &fromCompany)
	if err != nil {
		fmt.Println("Error unmarshalling account " + tr.FromCompany)
		return nil, errors.New("Error unmarshalling account " + tr.FromCompany)
	}

	var toCompany Account
	fmt.Println("Getting State on ToCompany " + tr.ToCompany)
	toCompanyBytes, err := stub.GetState(accountPrefix+tr.ToCompany);
	if err != nil {
		fmt.Println("Account not found " + tr.ToCompany)
		return nil, errors.New("Account not found " + tr.ToCompany)
	}

	fmt.Println("Unmarshalling tocompany")
	err = json.Unmarshal(toCompanyBytes, &toCompany)
	if err != nil {
		fmt.Println("Error unmarshalling account " + tr.ToCompany)
		return nil, errors.New("Error unmarshalling account " + tr.ToCompany)
	}

	// Check for all the possible errors
	ownerFound := false 
//	quantity := 0
	for _, owner := range cp.Owners {
		if owner.Company == tr.FromCompany {
			ownerFound = true
		//	quantity = owner.Quantity
		}
	}
	
	// If fromCompany doesn't own this paper
	if ownerFound == false {
		fmt.Println("The company " + tr.FromCompany + "doesn't own any of this paper");
		return nil, errors.New("The company " + tr.FromCompany + "doesn't own any of this paper");		
	} else {
		fmt.Println("The FromCompany does own this paper")
	}
	
	// If fromCompany doesn't own enought quantity of this paper
//commented by KD
	/*if quantity < tr.Quantity {
		fmt.Println("The company " + tr.FromCompany + "doesn't own enough of this paper");				
		return nil, errors.New("The company " + tr.FromCompany + "doesn't own enough of this paper");				
	} else {
		fmt.Println("The FromCompany owns enough of this paper")
	}
	*/
	amountToBeTransferred := float64(tr.Quantity) * cp.Par
//commented by KD	//amountToBeTransferred -= (amountToBeTransferred) * (cp.Discount / 100.0) * (float64(cp.Maturity) / 360.0)
	
	// If toCompany doesn't have enough cash to buy the papers
	// commented by KD //if toCompany.CashBalance < amountToBeTransferred {

	if fromCompany.CashBalance < amountToBeTransferred {
		fmt.Println("The company " + tr.ToCompany + "doesn't have enough cash to purchase the papers");			
		return nil, errors.New("The company " + tr.ToCompany + "doesn't have enough cash to purchase the papers");		
	} else {
		fmt.Println("The ToCompany has enough money to be transferred for this paper")
	}
	
	/* commented by KD
toCompany.CashBalance -= amountToBeTransferred
	fromCompany.CashBalance += amountToBeTransferred
*/
	fromCompany.CashBalance -= amountToBeTransferred
	toCompany.CashBalance += amountToBeTransferred

	toOwnerFound := false
/*commented by KD
	for key, owner := range cp.Owners {
		if owner.Company == tr.FromCompany {
			fmt.Println("Reducing Quantity from the FromCompany")
			cp.Owners[key].Quantity -= tr.Quantity
//			owner.Quantity -= tr.Quantity
		}
		if owner.Company == tr.ToCompany {
			fmt.Println("Increasing Quantity from the ToCompany")
			toOwnerFound = true
			cp.Owners[key].Quantity += tr.Quantity
//			owner.Quantity += tr.Quantity
		}
	}
	
	if toOwnerFound == false {
		var newOwner Owner
		fmt.Println("As ToOwner was not found, appending the owner to the Cheque")
		newOwner.Quantity = tr.Quantity
		newOwner.Company = tr.ToCompany
		cp.Owners = append(cp.Owners, newOwner)
	}
	*/
	fromCompany.AssetsIds = append(fromCompany.AssetsIds, tr.CUSIP)

	// Write everything back
	// To Company
	toCompanyBytesToWrite, err := json.Marshal(&toCompany)
	if err != nil {
		fmt.Println("Error marshalling the toCompany")
		return nil, errors.New("Error marshalling the toCompany")
	}
	fmt.Println("Put state on toCompany");
	err = stub.PutState(accountPrefix+tr.ToCompany, toCompanyBytesToWrite)
	if err != nil {
		fmt.Println("Error writing the toCompany back")
		return nil, errors.New("Error writing the toCompany back")
	}
		
	// From company
	fromCompanyBytesToWrite, err := json.Marshal(&fromCompany)
	if err != nil {
		fmt.Println("Error marshalling the fromCompany")
		return nil, errors.New("Error marshalling the fromCompany")
	}
	fmt.Println("Put state on fromCompany");
	err = stub.PutState(accountPrefix+tr.FromCompany, fromCompanyBytesToWrite)
	if err != nil {
		fmt.Println("Error writing the fromCompany back")
		return nil, errors.New("Error writing the fromCompany back")
	}
	
	// cp
	cpBytesToWrite, err := json.Marshal(&cp)
	if err != nil {
		fmt.Println("Error marshalling the cp")
		return nil, errors.New("Error marshalling the cp")
	}
	fmt.Println("Put state on Cheque");
	err = stub.PutState(cqPrefix+tr.CUSIP, cpBytesToWrite)
	if err != nil {
		fmt.Println("Error writing the cp back")
		return nil, errors.New("Error writing the cp back")
	}
	
	fmt.Println("Successfully completed Invoke");
	return nil, nil;
}

func (t *SimpleChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	//need one arg
	if len(args) < 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting ......")
	}

	if args[0] == "GetAllCheques" {
		fmt.Println("Getting all Cheques");
		allCPs, err := GetAllCheques(stub);
		if err != nil {
			fmt.Println("Error from GetAllCheques");
			return nil, err
		} else {
			allCPsBytes, err1 := json.Marshal(&allCPs)
			if err1 != nil {
				fmt.Println("Error marshalling allcps");
				return nil, err1
			}	
			fmt.Println("All success, returning allcps");
			return allCPsBytes, nil		 
		}
	} else if args[0] == "GetCheque" {
		fmt.Println("Getting particular cp");
		cp, err := GetCheque(args[1], stub);
		if err != nil {
			fmt.Println("Error Getting particular cp");
			return nil, err
		} else {
			cpBytes, err1 := json.Marshal(&cp)
			if err1 != nil {
				fmt.Println("Error marshalling the cp");
				return nil, err1
			}	
			fmt.Println("All success, returning the cp");
			return cpBytes, nil		 
		}
	} else if args[0] == "GetCompany" {
		fmt.Println("Getting the company");
		company, err := GetCompany(args[1], stub);
		if err != nil {
			fmt.Println("Error from getCompany");
			return nil, err
		} else {
			companyBytes, err1 := json.Marshal(&company)
			if err1 != nil {
				fmt.Println("Error marshalling the company");
				return nil, err1
			}	
			fmt.Println("All success, returning the company");
			return companyBytes, nil		 
		}
	} else {
		fmt.Println("Generic Query call");
		bytes, err := stub.GetState(args[0])

		if err != nil {
			fmt.Println("Some error happenend");
			return nil, errors.New("Some Error happened")
		}

		fmt.Println("All success, returning from generic");
		return bytes, nil		
	}
}

func (t *SimpleChaincode) Run(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("run is running " + function)
	
	if function == "issueCheque" {
		fmt.Println("Firing issueCheque");
		//Create an asset with some value
		return t.issueCheque(stub, args)
	} else if function == "encashCheque" {
		fmt.Println("Firing encashCheque");
		return t.transferPaper(stub, args)
	} else if function == "createAccounts" {
		fmt.Println("Firing createAccounts");
		return t.createAccounts(stub, args)
	}

	return nil, errors.New("Received unknown function invocation")
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Println("Error starting Simple chaincode: %s", err)
	}
}

//lookup tables for last two digits of CUSIP
var seventhDigit = map[int]string{
	1:  "A",
	2:  "B",
	3:  "C",
	4:  "D",
	5:  "E",
	6:  "F",
	7:  "G",
	8:  "H",
	9:  "J",
	10: "K",
	11: "L",
	12: "M",
	13: "N",
	14: "P",
	15: "Q",
	16: "R",
	17: "S",
	18: "T",
	19: "U",
	20: "V",
	21: "W",
	22: "X",
	23: "Y",
	24: "Z",
}

var eigthDigit = map[int]string{
	1:  "1",
	2:  "2",
	3:  "3",
	4:  "4",
	5:  "5",
	6:  "6",
	7:  "7",
	8:  "8",
	9:  "9",
	10: "A",
	11: "B",
	12: "C",
	13: "D",
	14: "E",
	15: "F",
	16: "G",
	17: "H",
	18: "J",
	19: "K",
	20: "L",
	21: "M",
	22: "N",
	23: "P",
	24: "Q",
	25: "R",
	26: "S",
	27: "T",
	28: "U",
	29: "V",
	30: "W",
	31: "X",
}
