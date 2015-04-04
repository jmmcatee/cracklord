cracklord.factory('QueueService', ['$http', function ($http) {
   var urlBase = '/api/queue'
   var QueueService = {};

   QueueService.reorder = function(orderArray) {
      var data = {};
      data.joborder = orderArray;
      return $http.put(urlBase, data);
   }

   return QueueService;
}]);