cracklord.factory('QueueService', ['$http', function ($http) {
   var QueueService = {};

   QueueService.reorder = function(orderArray) {
      var data = {};
      data.joborder = orderArray;
      return $http.put("/api/queue", data);
   }

   return QueueService;
}]);