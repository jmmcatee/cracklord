cracklord.factory('ResourceService', ['$resource', function ($resource) {
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
}]);

cracklord.service('ResourceList', ['ResourceService', function(ResourceService) {
   var list = null;
   var promise = ResourceService.query(
      function(data) {
         list = data;
      }
   );

   return { 
      promise: promise, 
      getList: function() {
         return list;
      }
   };
}]);