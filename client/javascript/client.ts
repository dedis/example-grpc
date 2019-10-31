import * as grpcWeb from 'grpc-web';
import  { CountClient } from './CountServiceClientPb'
import { CountRequest, CountResponse } from './count_pb'

var client = new CountClient('https://localhost:4000')

var request = new CountRequest()
request.setValue(0)

client.count(request, {'Access-Control-Allow-Origin': '*'}, (err: grpcWeb.Error, res: CountResponse) => {
    if (err) {
        console.error(err.code)
        console.error(err.message)
        return
    }
    console.log(res.getValue())
})