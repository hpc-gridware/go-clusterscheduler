/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2026 HPC-Gridware GmbH
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
*
************************************************************************/
/*___INFO__MARK_END__*/

package helper

import "errors"

// ErrNoSuchJob is the one sentinel for "the cluster does not know this
// job". The qstat and qalter packages re-export it under their own
// ErrNoSuchJob name, so errors.Is(err, helper.ErrNoSuchJob) and the
// package-local names match each other across packages.
var ErrNoSuchJob = errors.New("job does not exist")
