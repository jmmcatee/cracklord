cracklord.controller('JobsController', ['$scope', 'JobsService', 'growl', function JobsController($scope, JobsService, growl) {
	$scope.listreordered = false;
	$scope.now = Math.floor(Date.now() / 1000);
	$scope.jobactions = {};

	$scope.sortableOptions = {
		handle: '.draghandle',
		axis: 'y',
		update: function(e, ui) {
			$scope.listreordered = true;
		}
	};

	$scope.reorderConfirm = function() {

	}

	$scope.reorderCancel = function() {
		$scope.listreordered = false;
		$scope.loadJobs();
		growl.info("Job reorder was cancelled.")
	}

	$scope.loadJobs = function() {
		var jobs = JobsService.query(
			//Our success handler
			function(data) {
				$scope.listreordered = false;
				for(var i = 0; i < $scope.jobs.length; i++) {
					$scope.jobs[i].expanded = false;
				}
			},
			//Our error handler
			function(error) {
				switch (error.status) {
					case 400: growl.error("You sent bad data, check your input and if it's correct get in touch with us on github"); break;
					case 403: growl.warning("You're not allowed to do that..."); break;
					case 404: growl.error("That object was not found."); break;
					case 409: growl.error("The request could not be completed because there was a conflict with the existing resource."); break;
					case 500: growl.error("An internal server error occured while trying to add the resource."); break;
				}
			}
		);
		$scope.jobs = jobs;
	}
	$scope.loadJobs();
}]);

cracklord.directive('jobDetail', ['JobsService', 'growl', function jobDetail(JobsService, growl) {
	return {
		restrict: 'E',
		templateUrl: 'components/Jobs/jobsViewDetail.html',
		scope: {
			jobid: '@',
			visibility: '='
		},
		controller: function($scope) {
			// Mmmmmmmm.... Donut.
			$scope.processDonut = function() {
				$scope.donut = {};
				$scope.donut.labels = ['Cracked', 'Processed', 'Total'];

				var cracked = $scope.detail.crackedhashes;
				var processed = $scope.detail.totalhashes * $scope.detail.progress - cracked;
				var total = $scope.detail.totalhashes - processed;
				$scope.donut.data = [cracked, processed, total];

				$scope.donut.colors = [ '#5cb85c', '#337ab7', '#aaaaaa' ];
			};

			$scope.processLine = function() {
				$scope.line = {};
				$scope.line.series = [ $scope.detail.performancetitle ]; 
				$scope.line.data = [];
				$scope.line.data[0] = [];
				$scope.line.labels = [];
				$scope.line.options = {
					'pointDot': false,
					'showTooltips': false
				};
				$scope.line.colors = [
					'#d43f3a'
				]

				for(var time in $scope.detail.performancedata) {
					$scope.line.data[0].push($scope.detail.performancedata[time]);
					$scope.line.labels.push("");
				}
			}
		},
		link: function($scope, $element, $attrs) {
			$scope.$watch('visibility', function(newval, oldval) {
				if(newval === true) {
					JobsService.get({id: $scope.jobid}, 
						function success(data) {
							$scope.detail = data.job;
							$scope.processDonut();
							$scope.processLine();
							$element.parent().show();
							$element.find('.slider').slideDown();
						},
						function error(error) {
							growl.error("There was a problem loading job details.")
							$($element).find('div.slider').slideUp("slow", function() {
								$element.parent().hide();
							});
						}
					);
				} else {
					$($element).find('div.slider').slideUp("slow", function() {
						$element.parent().hide();
					});
				}
			});
		},
	}
}]);

cracklord.controller('CreateJobController', ['$scope', '$state', 'ToolsService', 'JobsService', 'growl', function CreateJobController($scope, $state, ToolsService, JobsService, growl) {
	$scope.formData = {};
	$scope.formData.params = {};

	$scope.toolChange = function() {
		var id = $scope.formData.tool.id;
		var tool = ToolsService.get({id: id}, 
			function(data) {
				$scope.tool = data.tool;
			}, 
			function(error) {
				growl.error("An error occured while trying to load tool information.");
			}
		);
	}

	$scope.processNewJobForm = function() {
		var newjob = new JobsService();

		newjob.toolid = $scope.formData.tool.id;
		newjob.name = $scope.formData.name;
		newjob.params = $scope.formData.params;
		
		JobsService.save(newjob, 
			function(data) {
				growl.success("Job successfully added");
				$state.transitionTo('jobs');
			}, 
			function(error) {
				switch (error.status) {
					case 400: growl.error("You sent bad data, check your input and if it's correct get in touch with us on github"); break;
					case 403: growl.warning("You're not allowed to do that..."); break;
					case 404: growl.error("That object was not found."); break;
					case 409: growl.error("The request could not be completed because there was a conflict with the existing resource."); break;
					case 500: growl.error("An internal server error occured while trying to add the resource."); break;
				}
			}
		);
	}	
}]);