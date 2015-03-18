cracklord.factory('ResourceService', function ($resource) {
   return $resource('/api/resources/:id', {id: '@id'}, {
   	query: {
   		isArray: true,
   		method: 'GET',
   		params: {},
   		transformResponse: function(data) {
   			var results = angular.fromJson(data);
   			return results.resources;
   		}	
   	},
      update: {
         method: 'PUT', 
         transformResponse: function(data) {
            var results = angular.fromJson(data);
            return results.resource;
         }
      }
   });
});
