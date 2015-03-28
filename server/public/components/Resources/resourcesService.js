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

cracklord.service('Resources', ['ResourceService', function(ResourceService) {
   this.list = null;

   this.promise = function () {
      var servers = ResourceService.query(
         function(data) {
            this.list = data;
         }
      );
   }

   this.reloadList();
   return this;
}]);