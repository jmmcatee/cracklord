cracklord.controller('JobsController', function JobsController($scope, Jobs){
	$scope.now = Math.floor(Date.now() / 1000);
	$scope.jobs = Jobs.list;

	$scope.sortableOptions = {
		handle: '.draghandle',
		update: function (e, ui) {
			console.log(e);
			console.log(ui);
		},
		axis: 'y'
	};
});