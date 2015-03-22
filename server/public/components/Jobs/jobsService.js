cracklord.factory('JobsService', function ($resource) {
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
         transformResponse: function(data) {
            var results = angular.fromJson(data);
            return results.job;
         }
      }
   });
});

cracklord.factory('QueueService', function($http) {
   return {
      reorder: function(jobs) {
          data = {};
          data.joborder = jobs;
          return $http.put('api/queue', data);
      }
   };
});

