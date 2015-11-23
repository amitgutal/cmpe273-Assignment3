package main

import  ("github.com/julienschmidt/httprouter"
		 "io/ioutil"
		 "net/http"
		 "encoding/json"
		 "fmt"
		 "gopkg.in/mgo.v2"
         "gopkg.in/mgo.v2/bson"
         "log"
         "strconv"
         "sort"
         "bytes")

// While testing the Code please enter the 
// Uber authentication code in the below 
// variable uber_server_auth_code and test the functionality

// const uber_server_auth_code = "YOUR SERVER CODE HERE"

const uber_server_auth_code = ""

func (a ByTotal) Len() int           { return len(a) }
func (a ByTotal) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTotal) Less(i, j int) bool { return a[i].Total < a[j].Total }

type Request struct{
	Starting_from_location_id string `json:"starting_from_location_id"`
	Location_ids [] string `json:"location_ids"` 
}

type LatLang struct{
	Coor Coordinate `json:"coordinate" bson:"coordinate"`
}

type Coordinate struct{
	Lat float64 `json:"lat" bson:"lat"`
	Lng float64 `json:"lng" bson:"lng"`
}


type UberDetails struct{
	Location_id string
	Cost float64
	Distance float64
	Duration float64
	Total float64
	Req_id string
}

type Ids struct{
	Id int  `bson:"id"`
}

type ByTotal []UberDetails


type Response struct{
	Id int `json:"id" bson:"id"`
	Status string `json:"status" bson:"status"`
	Starting_from_location_id string `json:"starting_from_location_id" bson:"starting_from_location_id"`
	Next_destination_location_id string `json:"next_destination_location_id,omitempty"`
	Best_route_location_ids [] string `json:"best_route_location_ids" bson:"best_route_location_ids"`
	Total_uber_costs float64 `json:"total_uber_costs"  bson:"total_uber_costs"`
	Total_uber_duration float64 `json:"total_uber_duration" bson:"total_uber_duration"`
	Total_distance float64 `json:"total_distance" bson:"total_distance"`
	Uber_wait_time_eta float64 `json:"uber_wait_time_eta,omitempty"`
	Index int `json:"-"`

}


func GetTrip(responseWriter http.ResponseWriter, request *http.Request,p httprouter.Params){
	
	params,_:= strconv.Atoi(p.ByName("tripid"))
	session, mongoError := mgo.Dial("mongodb://amit:1234@ds031407.mongolab.com:31407/cmpe273")
	if(mongoError!=nil){
		defer session.Close()
	}
	dbObject := session.DB("cmpe273").C("trips")
	var data Response
	mongoError = dbObject.Find(bson.M{"id":params}).Select(bson.M{"_id":0}).One(&data)
	if(mongoError!=nil){
		log.Printf("Querry Execution Error in Get Trip Details :  %s\n", mongoError) 
		fmt.Fprintln(responseWriter,mongoError)
				return
	}else{
		Jsonresult,_:=json.Marshal(data)
		fmt.Fprintln(responseWriter,string(Jsonresult))	
	}
}

func UberProductId(uber_start string, uber_end string)(string){
	
	var jsonInt interface{}
	
	response,uber_err:= http.Get("https://sandbox-api.uber.com/v1/products?latitude="+uber_start+"&longitude="+uber_end+"&access_token="+uber_server_auth_code+"")
	
	if uber_err!=nil{
		fmt.Println("Error in UberProduct -- Id",uber_err)
	}else{
		
		defer response.Body.Close()
		
		contents,uber2_err:= ioutil.ReadAll(response.Body)
		
		if(uber2_err!=nil){
			fmt.Println("Error in Accessing Uber API : ")
			fmt.Println(uber2_err)
		}
		json.Unmarshal(contents,&jsonInt)
		uber_product_id:= (jsonInt.(map[string] interface{})["products"]).([]interface{})[0].(map[string]interface{})["product_id"]
		return uber_product_id.(string)
	}
	return ""
}
func PostTrip(responseWriter http.ResponseWriter, request *http.Request,p httprouter.Params){
	
	var idResult Ids
	var input Request
	var LL LatLang
	var output Response
	var total float64
	var uberResponse [] UberDetails
	req,_:= ioutil.ReadAll(request.Body)
	json.Unmarshal(req,&input)
	session, err := mgo.Dial("mongodb://amit:1234@ds031407.mongolab.com:31407/cmpe273")
	if(err!=nil){
		defer session.Close()
	}
	dbObject := session.DB("cmpe273").C("users")
	oid := bson.ObjectIdHex(input.Starting_from_location_id)
	err = dbObject.FindId(oid).One(&LL)
	if(err!=nil){
	    log.Printf("In PUT REQUEST")
		log.Printf("Error in Query Execution in http PUT request  %s\n", err) 
		fmt.Fprintln(responseWriter,err)
				return
	}
	start_latitude:= strconv.FormatFloat(LL.Coor.Lat,'f',-1,64)
	start_longitude:= strconv.FormatFloat(LL.Coor.Lng,'f',-1,64)
	var end_latitude,end_longitude string
	output.Starting_from_location_id = input.Starting_from_location_id
	
	var jsonInt interface{}
	
	uberResponse= make([]UberDetails,len(input.Location_ids))
	
	for p, i:= range input.Location_ids{
		oid := bson.ObjectIdHex(i)
		err = dbObject.FindId(oid).One(&LL)
		if(err!=nil){
		log.Printf("Querry Execution Error in Post Trip Details  -->  ERROR  %s\n", err) 
		fmt.Fprintln(responseWriter,err)
				return
		}else{
			end_latitude = strconv.FormatFloat(LL.Coor.Lat,'f',-1,64)
			end_longitude = strconv.FormatFloat(LL.Coor.Lng,'f',-1,64)
			
			response,err:= http.Get("https://sandbox-api.uber.com/v1/estimates/price?start_latitude="+start_latitude+"&start_longitude="+start_longitude+"&end_latitude="+end_latitude+"&end_longitude="+end_longitude+"&access_token="+uber_server_auth_code+"")
			if err!=nil{
				fmt.Println("Error in accessing UBER SandBox API ",err)
			}else{
				defer response.Body.Close()
				contents,err:= ioutil.ReadAll(response.Body)
				if(err!=nil){
					fmt.Println(err)
				}
				json.Unmarshal(contents,&jsonInt)
				uberResponse[p].Cost = ((jsonInt.(map[string]interface{})["prices"]).([]interface{})[0].(map[string]interface{})["low_estimate"]).(float64)
				uberResponse[p].Duration =  ((jsonInt.(map[string]interface{})["prices"]).([]interface{})[0].(map[string]interface{})["duration"]).(float64)
				uberResponse[p].Distance = uberResponse[p].Distance + ((jsonInt.(map[string]interface{})["prices"]).([]interface{})[0].(map[string]interface{})["distance"]).(float64)		
				total = uberResponse[p].Cost * uberResponse[p].Duration
				uberResponse[p].Location_id = i
				uberResponse[p].Total = total
			}
		}
	}
	sort.Sort(ByTotal(uberResponse))
	output.Best_route_location_ids=make([]string,len(input.Location_ids))
	
	output.Best_route_location_ids[0]=uberResponse[0].Location_id
	
	output.Total_uber_costs = uberResponse[0].Cost
	
	output.Total_uber_duration =uberResponse[0].Duration
	
	output.Total_distance=uberResponse[0].Distance
	
	output.Index=0
	
	Array:=make([]string,len(uberResponse))
	 
	for a,arr:= range uberResponse{
	
		  Array[a] = arr.Location_id
	}
	temp:=1
	length:=len(Array)
	
	if length > 1{
		for j:=1; j < length ;j++{
			 uberResponse = BestRoute(Array,Array[0])
			 fmt.Print("UBER Routes")
			 fmt.Print(Array)
			 fmt.Println()
			 fmt.Print("UBER API Response - ")
			 fmt.Print(uberResponse)
			 fmt.Println()
			 if(len(uberResponse)!=0){
			      
				 output.Best_route_location_ids[j] =uberResponse[0].Location_id
				 output.Total_uber_costs = output.Total_uber_costs + uberResponse[0].Cost
				 output.Total_uber_duration = output.Total_uber_duration + uberResponse[0].Duration
				 output.Total_distance = output.Total_distance+ uberResponse[0].Distance
			 }else{
			 	output.Best_route_location_ids[j]=Array[0]
			 }
			 if(len(Array) > 1){
			 	// Array = Array[j:]
				Array = Array[temp:]
			 }
			 
		}
	}
	
	tempArray :=[] string{Array[0],input.Starting_from_location_id}
	uberResponse = BestRoute(tempArray,Array[0])
	output.Total_uber_costs = output.Total_uber_costs + uberResponse[0].Cost
    output.Total_uber_duration = output.Total_uber_duration + uberResponse[0].Duration
    output.Total_distance = output.Total_distance+ uberResponse[0].Distance
	
	output.Status="planning"
	o := session.DB("cmpe273").C("trips")
	idResult.Id = 0
	count,_:=o.Count()
	if(count > 0){
		err := o.Find(nil).Select(bson.M{"id":1000}).Sort("-id").One(&idResult)
		if(err!=nil){
		log.Printf("Posting Trip Details")
			log.Printf("Querry Execution Error in Post Trip Details %s\n", err) 
			fmt.Fprintln(responseWriter,err)
			return 
		}
		output.Id = idResult.Id + 100
        err = o.Insert(output)
        if err != nil {
                log.Fatal(err)
        }
        result,_:=json.Marshal(output)
		fmt.Fprintln(responseWriter,string(result))
	}else{
		output.Id = idResult.Id + 100
        err = o.Insert(output)
        if err != nil {
                log.Fatal(err)
        }
        result,_:=json.Marshal(output)
		fmt.Fprintln(responseWriter,string(result))	
	}
}


func BestRoute(Array [] string,Starting_from_location_id string) []UberDetails{
	var data LatLang
	var total float64
	uberResponse:=make([] UberDetails,len(Array)-1)
	session, err := mgo.Dial("mongodb://amit:1234@ds031407.mongolab.com:31407/cmpe273")
	if(err!=nil){
		defer session.Close()
	}
	c := session.DB("cmpe273").C("users")
	oid := bson.ObjectIdHex(Starting_from_location_id)
	err = c.FindId(oid).One(&data)
	if(err!=nil){
		log.Printf("Querry Execution Error in Sorting Details -->>  %s\n", err) 
	}
	start_latitude:= strconv.FormatFloat(data.Coor.Lat,'f',-1,64)
	start_longitude:= strconv.FormatFloat(data.Coor.Lng,'f',-1,64)
	var end_latitude,end_longitude string
	var jsonInt interface{}
	
	for p := 1;p < len(Array);p++{
		oid := bson.ObjectIdHex(Array[p])
		err = c.FindId(oid).One(&data)
		if(err!=nil){
		log.Printf("RunQuery : ERROR : %s\n", err) 
		}else{
			end_latitude = strconv.FormatFloat(data.Coor.Lat,'f',-1,64)
			end_longitude = strconv.FormatFloat(data.Coor.Lng,'f',-1,64)
			response,err:= http.Get("https://sandbox-api.uber.com/v1/estimates/price?start_latitude="+start_latitude+"&start_longitude="+start_longitude+"&end_latitude="+end_latitude+"&end_longitude="+end_longitude+"&access_token="+uber_server_auth_code+"")
			if err!=nil{
				fmt.Println("Error in accessing UBER API ",err)
			}else{
				defer response.Body.Close()
				contents,err:= ioutil.ReadAll(response.Body)
				if(err!=nil){
					fmt.Println(err)
				}
				json.Unmarshal(contents,&jsonInt)
				uberResponse[p-1].Cost = ((jsonInt.(map[string]interface{})["prices"]).([]interface{})[0].(map[string]interface{})["low_estimate"]).(float64)
				uberResponse[p-1].Duration =  ((jsonInt.(map[string]interface{})["prices"]).([]interface{})[0].(map[string]interface{})["duration"]).(float64)			
				uberResponse[p-1].Distance = ((jsonInt.(map[string]interface{})["prices"]).([]interface{})[0].(map[string]interface{})["distance"]).(float64)		
				total = uberResponse[p-1].Cost * uberResponse[p-1].Duration
				uberResponse[p-1].Location_id = Array[p]
				uberResponse[p-1].Total = total
				
			}
		}
	}
	sort.Sort(ByTotal(uberResponse))
	return uberResponse
}

func PutTrip(respWriter http.ResponseWriter, request *http.Request,p httprouter.Params){
	
	var request_id string
	var eta float64
	var status string
	tripparameters,_:= strconv.Atoi(p.ByName("tripid"))	
	
	session, sessionerr := mgo.Dial("mongodb://amit:1234@ds031407.mongolab.com:31407/cmpe273")
	if(sessionerr!=nil){
		defer session.Close()
	}
	db_connection_object := session.DB("cmpe273").C("trips")
	var data Response
	sessionerr = db_connection_object.Find(bson.M{"id":tripparameters}).Select(bson.M{"_id":0}).One(&data)
	if(sessionerr!=nil){
		log.Printf("Querry Execution Error in Put Request Details %s\n", sessionerr) 
		fmt.Fprintln(respWriter,sessionerr)
				return
	}
	index:= data.Index
	
	var tempIn string
	if(index==0){
		tempIn = data.Starting_from_location_id
	}else{
		tempIn = data.Best_route_location_ids[index-1]
	}
	if(index <len(data.Best_route_location_ids)){
		if(data.Starting_from_location_id!=data.Best_route_location_ids[index]){
			request_id,eta,status = GetUberDetails(tempIn,data.Best_route_location_ids[index])
			data.Next_destination_location_id = data.Best_route_location_ids[index]
			data.Uber_wait_time_eta = eta
			data.Status = status
			jsonStr,_:= json.Marshal(map[string] interface{}{"status":"completed"})
			req,err := http.NewRequest("PUT","https://sandbox-api.uber.com/v1/sandbox/requests/"+request_id,bytes.NewBuffer(jsonStr))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization","Bearer "+uber_server_auth_code)
			client := &http.Client{}
		    resp, err := client.Do(req)
		    if err!=nil{
				fmt.Println("Error in processing Client Request in http PUT",err)
			}else{
				defer resp.Body.Close()
				result,_:=json.Marshal(data)
				fmt.Fprintln(respWriter,string(result))	
				tempIn = data.Best_route_location_ids[index]
				index++
				err = db_connection_object.Update(bson.M{"id":tripparameters},bson.M{"$set":bson.M{"index":index}})
			}	
		}else{
			fmt.Fprintln(respWriter,"Location IDs are same.Please place another one in put the http PUT request")
			index++
			sessionerr = db_connection_object.Update(bson.M{"id":tripparameters},bson.M{"$set":bson.M{"index":index}})
			}
	}else{
		
		// fmt.Fprintln(respWriter,"Destination successfully Reached.")
		// index=0
		// sessionerr = db_connection_object.Update(bson.M{"id":tripparameters},bson.M{"$set":bson.M{"index":index}})
		  
		    request_id,eta,status = GetUberDetails(tempIn,data.Starting_from_location_id)
			data.Next_destination_location_id = data.Starting_from_location_id
			data.Uber_wait_time_eta = eta
			data.Status = status
			jsonStr,_:= json.Marshal(map[string] interface{}{"status":"completed"})
			req,err := http.NewRequest("PUT","https://sandbox-api.uber.com/v1/sandbox/requests/"+request_id,bytes.NewBuffer(jsonStr))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization","Bearer "+uber_server_auth_code)
			client := &http.Client{}
		    resp, err := client.Do(req)
		    if err!=nil{
				fmt.Println("Error:",err)
			}else{
				defer resp.Body.Close()
				result,_:=json.Marshal(data)
				fmt.Fprintln(respWriter,string(result))	
				index=0
				err = db_connection_object.Update(bson.M{"id":tripparameters},bson.M{"$set":bson.M{"index":index}})
				
			}
		
	}

}

func GetUberDetails(uber_start string, end string)(string,float64,string){
	var jsonInt interface{}
	var LL LatLang
	mongosession, mongoerr := mgo.Dial("mongodb://amit:1234@ds031407.mongolab.com:31407/cmpe273")
	if(mongoerr!=nil){
		defer mongosession.Close()
	}

	db_connection := mongosession.DB("cmpe273").C("users")
	
	oid := bson.ObjectIdHex(uber_start)
	
    mongoerr = db_connection.FindId(oid).One(&LL)
	
	if(mongoerr!=nil){
		
		log.Printf("Querry Execution Error in get Details Function %s ", mongoerr) 
	}
	start_latitude:= strconv.FormatFloat(LL.Coor.Lat,'f',-1,64)
	
	start_longitude:= strconv.FormatFloat(LL.Coor.Lng,'f',-1,64)
	if(mongoerr!=nil){
	
	    	log.Printf("Querry Execution Error in Function %s ", mongoerr) 
	} else { 
	        //  
	  }
	var end_latitude,end_longitude string
	oid1 := bson.ObjectIdHex(end)
	mongoerr = db_connection.FindId(oid1).One(&LL)
	if(mongoerr!=nil){
		
	   log.Printf("Querry Execution Error in get Details Function 2  %s\n", mongoerr) 
	
	}else{
		
		end_longitude = strconv.FormatFloat(LL.Coor.Lng,'f',-1,64)
		end_latitude = strconv.FormatFloat(LL.Coor.Lat,'f',-1,64)
		product_id := UberProductId(start_latitude,start_longitude)
		stringJSON,_:= json.Marshal(map[string] interface{}{
		"product_id":product_id,"start_latitude":start_latitude,"start_longitude":start_longitude,"end_latitude":end_latitude,"end_longitude":end_longitude})
		req,err := http.NewRequest("POST","https://sandbox-api.uber.com/v1/requests",bytes.NewBuffer(stringJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization","Bearer "+uber_server_auth_code)
		client := &http.Client{}
		if err!=nil{
			fmt.Println("Error in Requesting UBER API in getDetails METHOD",err)
		}
	    resp, resp_err := client.Do(req)
	    if resp_err!=nil{
			fmt.Println("Error in Requesting UBER API in getDetails METHOD",resp_err)
		}else{
			defer resp.Body.Close()
			contents,resp_err:= ioutil.ReadAll(resp.Body)
			
			if(resp_err!=nil){
				fmt.Println(resp_err)
			}else{	
				json.Unmarshal(contents,&jsonInt)
				fmt.Println(jsonInt)
				request_id:= jsonInt.(map[string]interface{})["request_id"]
				eta:= jsonInt.(map[string]interface{})["eta"]
				status:= jsonInt.(map[string]interface{})["status"]
				return request_id.(string),eta.(float64),status.(string)
			}
		}
	}
	return "",0.0,""
}

func main() {
	muxer := httprouter.New()
    muxer.GET("/trips/:tripid",GetTrip) 
    muxer.PUT("/trips/:tripid/request",PutTrip)
	muxer.POST("/trips",PostTrip)
    server := http.Server{
            Addr:        "0.0.0.0:8080",
            Handler: muxer,
    }
    server.ListenAndServe()
}