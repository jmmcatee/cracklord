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