cracklord.controller('JobsController', function JobsController($scope, JobsService, growl) {
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
				growl.success("Jobs successfully loaded.");
			},
			//Our error handler
			function(error) {
				growl.error("An error occured while trying to load jobs.");
			}
		);
		$scope.jobs = jobs;
	}
	$scope.loadJobs();
});

cracklord.directive('jobDetail', function jobDetail(JobsService, growl) {
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
});

cracklord.controller('CreateJobController', function CreateJobController($scope, $state, ToolsService, JobsService, growl) {
	$scope.formData = {};
	$scope.formData.params = {};

	$scope.toolChange = function() {
		var toolid = $scope.formData.tool.toolid;
		var tool = ToolsService.get({toolid: toolid}, 
			function(data) {
				$scope.tool = data;
			}, 
			function(error) {
				growl.error("An error occured while trying to load tool information.");
			}
		);
	}

	$scope.processNewJobForm = function() {
		var newjob = new JobsService();

		newjob.toolid = $scope.formData.tool.toolid;
		newjob.name = $scope.formData.name;
		newjob.params = $scope.formData.params;
		
		JobsService.save(newjob, 
			function(data) {
				growl.success("Job successfully added");
				$state.transitionTo('jobs');
			}, 
			function(error) {
				growl.error("An error occured while trying to save the job.");
			}
		);
	}	
});

cracklord.animation('.job-detail', function() {
	return {
		enter: function(element, done) {
		},
		leave: function(element, done) {
		}
	};	
})