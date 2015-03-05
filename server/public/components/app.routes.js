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
		})
		.state('createjob', {
			url:'/jobs/new',
			templateUrl: 'components/Jobs/newJob.html',
			controller: 'CreateJobController'
		})
		.state('createjob.tools', {
			url:"/tools",
			templateUrl: 'components/Jobs/newJob.tools.html',
		})
		.state('createjob.details', {
			url:"/details",
			templateUrl: 'components/Jobs/newJob.details.html',
		});
});