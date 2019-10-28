//  Copyright 2019 U.C. Berkeley RISE Lab
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

#include <assert.h>
#include <grpcpp/grpcpp.h>
#include <google/protobuf/empty.pb.h>

#include <fstream>
#include <iostream>
#include <memory>
#include <string>
#include <vector>
#include <typeinfo>

#include "objstore.grpc.pb.h"
#include "client/kvs_client.hpp"
#include "yaml-cpp/yaml.h"

using grpc::Server;
using grpc::ServerBuilder;
using grpc::ServerContext;
using grpc::Status;
using grpc::StatusCode;
using objstore::ObjectStore;
using objstore::CreateBucketRequest;
using objstore::ListBucketRequest;
using objstore::ListBucketResponse;
using objstore::GetRequest;
using objstore::GetResponse;
using objstore::PutRequest;
using objstore::DeleteBucketRequest;
using objstore::DeleteRequest;
using google::protobuf::Empty;

unsigned kRoutingThreadCount;

ZmqUtil zmq_util;
ZmqUtilInterface *kZmqUtil = &zmq_util;

Status statusHandler(KeyResponse response) {
  AnnaError error = response.tuples()[0].error();
  switch (error) {
    case AnnaError::NO_ERROR:
              return Status(StatusCode::OK, "Success!");
    case AnnaError::KEY_DNE:
              return Status(StatusCode::NOT_FOUND, "Not Found!");
    case AnnaError::WRONG_THREAD:
              return Status(StatusCode::INTERNAL, "Wrong Thread!");
    case AnnaError::NO_SERVERS: case AnnaError::TIMEOUT:
              return Status(StatusCode::UNAVAILABLE, "Unavailable!");
    case AnnaError::LATTICE:
              return Status(StatusCode::INVALID_ARGUMENT, "Incorrect Lattice!");
    default:  return Status(StatusCode::UNKNOWN, "Unknown Failure!");
  }
}

Status put_request(KvsClientInterface *client, const Key &key,
                   const string &payload) {
  string rid = client->put_async(
                        key, payload, LatticeType::SET);
  vector<KeyResponse> responses = client->receive_async();
  while (responses.size() == 0) {
    responses = client->receive_async();
  }
  KeyResponse response = responses[0];

  if (response.response_id() != rid) {
    // rarely happens
    std::cout << "Invalid response: ID did not match request ID!"
              << std::endl;
  }

  return statusHandler(response);
}



KeyResponse get_request(KvsClientInterface *client, Key key) {
  client->get_async(key);

  vector<KeyResponse> responses = client->receive_async();
  while (responses.size() == 0) {
    responses = client->receive_async();
  }

  if (responses.size() > 1) {
    // shouldn't happen unless you request put_all/get_all
    std::cout << "Error: received more than one response" << std::endl;
  }

  KeyResponse response = responses[0];
  // assert(response.tuples(0).lattice_type() == LatticeType::SET);

  return response;
}

// Logic and data behind the server's behavior.
class LocalObjStoreImpl final : public ObjectStore::Service {
  // essentially upload a kv pair <bucket name, empty set>,
  // will overwrite it if the key exists.
  Status createBucket(ServerContext* context,
                    const CreateBucketRequest* request, Empty* reply) override {
    Key key = request->bucketname();
    SetLattice<std::string> val;  // empty set lattice
    val.insert("");

    return put_request(this->getAnnaClient(), key, serialize(val));
  }

  Status listBucket(ServerContext* context,
        const ListBucketRequest* request, ListBucketResponse* reply) override  {
    Key key = request->bucketname();
    KeyResponse response = get_request(this->getAnnaClient(), key);
    
    Status status = statusHandler(response);
    if (!status.ok())
      return status;

    SetLattice<std::string> set_lattice
              = deserialize_set(response.tuples(0).payload());
    auto filenames = set_lattice.reveal();

    for (auto itr = filenames.begin(); itr != filenames.end(); ++itr) {
      reply->add_objectname(*itr);
    }
    return status;
  }

  Status deleteBucket(ServerContext* context,
                    const DeleteBucketRequest* request, Empty* reply) override {
    Key key = request->bucketname();
    // check if bucket is empty
    KeyResponse response = get_request(this->getAnnaClient(), key);
    Status status = statusHandler(response);
    if (!status.ok())
      return status;
    SetLattice<std::string> set_lattice =
        deserialize_set(response.tuples(0).payload());
    if (set_lattice.size().reveal() > 1) // a bucket always have an empty objectname
      return Status(StatusCode::PERMISSION_DENIED,
                    "Cannot delelet a nonempty bucket!");
    // delete the bucket
    return put_request(this->getAnnaClient(), key, string());
  }

  Status get(ServerContext* context,
             const GetRequest* request, GetResponse* reply) override {
    Key key = request->bucketname();
    // get the pair <bucketname, set of objectnames>
    KeyResponse response = get_request(this->getAnnaClient(), key);
    Status status = statusHandler(response);
    if (!status.ok())
      return status;

    SetLattice<std::string> set_lattice =
        deserialize_set(response.tuples(0).payload());
    std::unordered_set<string> filenames = set_lattice.reveal();
    // check if the object exists in the bucket
    string filename = request->objectname();
    string objname = key + "/" + filename;
    if (filenames.find(objname) == filenames.end()) {
      return Status(StatusCode::NOT_FOUND, "Object doesn't exist!");
    }
    // get the object
    response = get_request(this->getAnnaClient(), objname);
    status = statusHandler(response);
    if (!status.ok())
      return status;
    // extract the object from the response
    LWWPairLattice<std::string> lww_lattice =
        deserialize_lww(response.tuples(0).payload());
    std::string data = lww_lattice.reveal().value;
    reply->set_data(data);

    return status;
  }

  Status put(ServerContext* context,
             const PutRequest* request, Empty* reply) override {
    Key key = request->bucketname();
    // get the pair <bucketname, set of object names>
    KeyResponse response = get_request(this->getAnnaClient(), key);
    Status status = statusHandler(response);
    if (!status.ok())  // should fail if no such keyVal exists
      return status;
    SetLattice<std::string> set_lattice =
        deserialize_set(response.tuples(0).payload());
    std::unordered_set<string> filenames = set_lattice.reveal();
    // first add the object name to the bucket
    string filename = request->objectname();
    string objname = key + "/" + filename;
    set_lattice.insert(objname);
    status = put_request(this->getAnnaClient(), key,
                                serialize(set_lattice));
    // second, put the object
    string data = request->data();
    LWWPairLattice<std::string> val(
        TimestampValuePair<std::string>(generate_timestamp(0), data));
    
    string rid = this->getAnnaClient()->put_async(
                        objname, serialize(val), LatticeType::LWW);
    vector<KeyResponse> responses = this->getAnnaClient()->receive_async();
    while (responses.size() == 0) {
      responses = this->getAnnaClient()->receive_async();
    }
    response = responses[0];

    if (response.response_id() != rid) {
      // rarely happens
      std::cout << "Invalid response: ID did not match request ID!"
                << std::endl;
    }

    return statusHandler(response);
    // return put_request(this->getAnnaClient(), objname, serialize(val));
  }

  Status deleteObject(ServerContext* context,
                const DeleteRequest* request, Empty* reply) override {
    Key key = request->bucketname();
    // get the pair <bucketname, set of object names>
    KeyResponse response = get_request(this->getAnnaClient(), key);
    Status status = statusHandler(response);
    if (!status.ok())
      return status;
    SetLattice<std::string> set_lattice =
        deserialize_set(response.tuples(0).payload());
    std::unordered_set<string> filenames = set_lattice.reveal();
    // first remove the objectname from the bucket
    string filename = request->objectname();
    string objname = key + "/" + filename;
    int num = filenames.erase(objname);
    if (num == 0)  // objectname doesn't exist in the bucket
      return Status(StatusCode::NOT_FOUND, "Not Found!");
    
    SetLattice<std::string> emptySet;
    status = put_request(this->getAnnaClient(), key, serialize(emptySet));
    if (!status.ok())
      return status;
    
    SetLattice<std::string> val(filenames);
    status = put_request(this->getAnnaClient(), key, serialize(val));
    if (!status.ok())
      return status;
    // finally delete the object
    string emptyString = string();
    LWWPairLattice<std::string> emp(
        TimestampValuePair<std::string>(generate_timestamp(0), emptyString));
    string rid = this->getAnnaClient()->put_async(
                        objname, serialize(emp), LatticeType::LWW);
    vector<KeyResponse> responses = this->getAnnaClient()->receive_async();
    while (responses.size() == 0) {
      responses = this->getAnnaClient()->receive_async();
    }
    response = responses[0];

    if (response.response_id() != rid) {
      // rarely happens
      std::cout << "Invalid response: ID did not match request ID!"
                << std::endl;
    }

    return statusHandler(response);
    //return put_request(this->getAnnaClient(), objname, string());
  }

  KvsClientInterface *annaClient;
  inline KvsClientInterface *getAnnaClient(void) { return annaClient; }

 public:
    explicit LocalObjStoreImpl(KvsClientInterface *client) {
      annaClient = client;
    }
};

void RunServer(KvsClientInterface *client) {
  std::string server_address("127.0.0.1:50051");
  LocalObjStoreImpl service(client);

  ServerBuilder builder;
  // Listen on the given address without any authentication mechanism.
  builder.AddListeningPort(server_address, grpc::InsecureServerCredentials());
  // Register "service" as the instance through which we'll communicate with
  // clients. In this case, it corresponds to an *synchronous* service.
  builder.RegisterService(&service);
  // Set mxax message size to INT_MAX
  builder.SetMaxReceiveMessageSize(INT_MAX);
  // Finally assemble the server.
  std::unique_ptr<Server> server(builder.BuildAndStart());
  std::cout << "Server listening on " << server_address << std::endl;

  // Wait for the server to shutdown. Note that some other thread must be
  // responsible for shutting down the server for this call to ever return.
  server->Wait();
}

int main(int argc, char *argv[]) {
  if (argc < 2 || argc > 2) {
    std::cerr << "Usage: " << argv[0] << " conf-file" << std::endl;
    return 1;
  }

  // read the YAML conf
  YAML::Node conf = YAML::LoadFile(argv[1]);
  kRoutingThreadCount = conf["threads"]["routing"].as<unsigned>();

  YAML::Node user = conf["user"];
  Address ip = user["ip"].as<Address>();

  vector<Address> routing_ips;
  if (YAML::Node elb = user["routing-elb"]) {
    routing_ips.push_back(elb.as<string>());
  } else {
    YAML::Node routing = user["routing"];
    for (const YAML::Node &node : routing) {
      routing_ips.push_back(node.as<Address>());
    }
  }

  vector<UserRoutingThread> threads;
  for (Address addr : routing_ips) {
    for (unsigned i = 0; i < kRoutingThreadCount; i++) {
      threads.push_back(UserRoutingThread(addr, i));
    }
  }

  KvsClient client(threads, ip, 0, 10000);

  RunServer(&client);
}
