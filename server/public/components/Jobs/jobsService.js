cracklord.factory('JobsService', ['$resource', function ($resource) {
   return $resource('/api/jobs/:id', {id: '@id'}, {
   	query: {
   		isArray: true,
   		method: 'GET',
   		params: {},
   		transformResponse: function(data) {
   			var results = angular.fromJson(data);
   			return results.jobs;
   		}	
   	},
    update: {
      method: 'PUT', 
      transformResponse: function(data, headersGetter, status) {
        if(status >= 200 && status <= 400) {
          var results = angular.fromJson(data);
          return results.job;
        } else {
          return data;
        }
      }
    }
  });
}]);

cracklord.factory('QueueService', ['$http', function($http) {
  return {
    reorder: function(jobs) {
      data = {};
      data.joborder = jobs;
      return $http.put('api/queue', data);
    }
  };
}]);

