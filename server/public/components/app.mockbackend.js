angular.module('cracklord').run(function($httpBackend, UserDataModel, JobsDataModel, ToolsDataModel, ResourcesDataModel) {
    $httpBackend.whenPOST(/\/api\/login/).passThrough();
    $httpBackend.whenGET(/\/api\/logout/).passThrough();

    $httpBackend.whenGET('/api/queue').respond(function(method, url, data) {
        var jobs = JobsDataModel.query();
    });

    $httpBackend.whenGET(/\/tools\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/).respond(function(method, url, data) {
        var id = url.split('/')[3];
        var tool = ToolsDataModel.read(id);

        if(tool != false) {
            tool["status"] = 200;
            tool["message"] = "OK";
            return [200, {'status': 200, 'message': 'OK', 'tool': tool}, {}];
        } else {
            return [404, {"status": 404, "message": "Tool "+id+" not found."}, {}];
        }
    });

    $httpBackend.whenGET('/api/tools').respond(function(method, url, data) {
        var tools = ToolsDataModel.query();
        var returninfo = { "status": 200, "message": "OK", "tools": tools};
        return [200, returninfo, {}];
    });

    /*
    $httpBackend.whenGET('/api/resources').respond(function(method, url, data) {
        var resources = ResourcesDataModel.query();
        var returninfo = { "status": 200, "message": "OK", "resources": resources};
        return [200, returninfo, {}];
    });

    $httpBackend.whenGET(/\/resources\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/).respond(function(method, url, data) {
        var id = url.split('/')[3];
        var resource = ResourcesDataModel.read(id);

        if(resource != false) {
            return [200, {'status': 200, 'message': 'OK', 'resource': resource} , {}];
        } else {
            return [404, {"status": 404, "message": "Resource "+id+" not found."}, {}];
        }
    });*/
    $httpBackend.whenGET('/api/resources').passThrough();
    $httpBackend.whenGET(/\/resources\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/).passThrough();
    $httpBackend.whenPOST('/api/resources').passThrough();
    $httpBackend.whenPUT(/\/resources\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/).passThrough();
    $httpBackend.whenDELETE(/\/resources\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/).passThrough();

    $httpBackend.whenGET('/api/jobs').respond(function(method, url, data) {
        var jobs = JobsDataModel.query();
        var returninfo = { "status": 200, "message": "OK", "jobs": jobs};
        return [200, returninfo, {}];
    });

    $httpBackend.whenGET(/\/jobs\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/).respond(function(method, url, data) {
        var id = url.split('/')[3];
        
        var job = JobsDataModel.read(id);

        if(job != false) {
            return [200, {'status': 200, "message": "OK", "job": job}, {}];
        } else {
            return [404, {"status": 404, "message": "Job "+id+" not found."}, {}];
        }
    });

    $httpBackend.whenPOST('/api/jobs').respond(function(method, url, data) {
        var params = angular.fromJson(data);
        var id = JobsDataModel.create(params);
        
        return [201, {"status": 201, "message": "Job "+id+" successfully created.", "id": id}, {}];
    });

    $httpBackend.whenPUT(/\/jobs\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/).respond(function(method, url, data) {
        var params = angular.fromJson(data);
        var id = url.split('/')[3];
        var result = JobsDataModel.update(id, params);
       
        if(!result) {
            return [404, { "status": 404, "message": "Job not found" }, {}];
        } else {
            return [200, {"status": 200, "message": "OK", "job": result}, {}];
        }
    });
    
    $httpBackend.whenDELETE(/\/jobs\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/).respond(function(method, url, data) {
        var id = url.split('/')[3];
        var result = JobsDataModel.delete(id);
        
        if(result == true) {
            return [200, {"status": 200, "message": "OK"}, {}];
        } else {
            return [404, {"status": 404, "message": "Job not found"}, {}];
        }
    });    

    $httpBackend.whenGET(/components/).passThrough();
});

