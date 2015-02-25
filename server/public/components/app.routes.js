cracklord.config(function($stateProvider, $urlRouterProvider) {
	$urlRouterProvider.otherwise('/jobs');
	$stateProvider
		.state('jobs', {
			url:'/jobs',
			templateUrl: 'components/Jobs/jobsView.html',
		})
	.state('resources', {
		url:'/resources',
			templateUrl: 'components/Resources/resourcesView.html',
		});
});