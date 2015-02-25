cracklord.controller('ResourcesController', function ResourcesController($scope, Resources){
    $scope.resources = Resources.list;
});