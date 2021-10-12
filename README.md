# delivery-agent
Go application that delivers http responses

Services used: go1.17.2 and segmentio/kafka-go

Log file location: /var/log/delivery-agent.txt

Sample incoming data from Kafka Consumer:

'{"mascot":"Gopher","location":"https:\/\/blog.golang.org\/gopher\/gopher.png","data": {"mascot":"Gopher","location":"https:\/\/blog.golang.org\/gopher\/gopher.png"}, "endpoint_method":"GET","endpoint_url":"http:\/\/sample_domain_endpoint.com\/data?title={mascot}&image={location}&foo={bar}","start_time":"2021.10.12 03:22:04.199288"}


Delivery Agent Workflow:
1. Open file for logging delivery events
2. Create Kafka Consumer
3. When a message comes through consumer:
   - Decode message using Message Struct
   - Process the url in from "endpoint_url" in message (replace {xxx} with corresponding values in "data" object
   - Create a datetime to represent the delivery time
   - Calculate the milliseconds between when the HTTP request was processed by PHP and when it was processed by the delivery agent
   - Make the a HTTP request using the processed url and "endpoint_method". This will attempt the request 3 times before logging an error do the log file from #1
   - Otherwise add a log to log file (see below for file format)
   
Log File Format:
-------------------------
<log timestamp> <delivery timestamp> <delivery total time>  <response timestamp>  <total response time> <response status code> <response body>
  
 - log timestamp -> Time that the request response or error was added to the event log
 - delivery timestamp -> Time when the original request was processed by the delivery agent
 - delivery total time -> Total time it took for the original HTTP request to be ingested, queued, unqueued and processed
 - response timestamp -> Time when delivery response was received by the delivery agent
 - total response time -> Total time it took for the original HTTP request to be ingested, queued, unqueued, processed, send new request and receive response
 - response status code -> HTTP Status code received back from new HTTP request
 - response body -> Payload received back from new HTTP request

 Sample Success Entry:
 ------------------------
 
2021/10/12 04:03:42 2021-10-12 04:03:41.057028322 +0000 UTC   2   2021-10-12 04:03:42.361965545 +0000 UTC   1306   200   <!doctype html><html itemscope="" itemtype="http://schema.org/WebPage" lang="en"><head>
  
Sample Failed Entry:
------------------------
  
2021/10/12 03:50:25 Failed to get response from url:  http://sample_domain_endpoint.com/data?title=Puppy&image=https://blog.golang.org/puppy/puppy.png&foo=null  using method:  GET
  

Troubleshooting:
------------------------
Error log location: /var/log/delivery-agent.log
Event log location: /var/log/delivery-agent.txt

If you don't see you request response in the event log, check for errors in the error log. 
If there is a failed entry for your request in the event log, this typically means that a network error occurred.
  


