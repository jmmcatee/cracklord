cracklord.factory('ResourceManagersService', ['$resource', function ($resource) {
   return $resource('/api/resourcemanagers/:id', {id: '@id'}, {
   	query: {
   		isArray: true,
   		method: 'GET',
   		params: {},
   		transformResponse: function(data) {
   			var results = angular.fromJson(data);
   			return results.resourcemanagers;
   		}	
   	},
   });
}]);

cracklord.service('ResourceManagers', ['ResourceManagersService', '$filter' function(ResourceManagersService, $filter) {
   var managers = {};
   managers.list = [];

   managers.load = function() {
      return ResourceManagersService.query(
         function(data) {
            angular.copy(data, managers.list)
         }
      );
   }

   managers.get = function(id) {
      return ResourceManagersService.get({id: id});
   }

   return managers;
}]);