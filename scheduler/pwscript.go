package scheduler

import "fmt"

func createPwScript(src, dest, folder, appTitle string, backupLimit, toastExpirationTimeInMinutes uint8, overwrite bool) string {
	if backupLimit >= 10 {
		backupLimit = 0
	}
	return fmt.Sprintf(`
	function Copy-Folder($src, $dest) {
		xcopy $src $destPath /e /y /i /h /o /k /x;
		return
	}
	function Rename-Backup($destPath, $folder) {
		try {
			$newFolderName = $folder + '-' + $(Get-Date -Format 'yyyyMMdd_HHmmss');
			Rename-Item  $destPath $newFolderName;
		} catch {
			return $false
		}
		return $true
	}
	function Remove-Backup($backupLimit, $dest, $folderName) {
		$searchPattern = $dest + '\' + $folderName + '-' + '20[0-9][0-9][0-9][0-9][0-9][0-9]_[0-9][0-9][0-9][0-9][0-9][0-9]';
		$backupFolderArray = Get-ChildItem -Path $searchPattern | Sort-Object CreationTime;
		if(($backupFolderArray.length - $backupLimit) -GT 0) {
			try {
				$shouldDelete = $backupFolderArray.length - $backupLimit;
				for ($index = 0; $index -lt $backupFolderArray.length - $backupLimit; $index++) {
					$deletePath = $dest + '\' + $backupFolderArray[$index].Name
					$partiallyDeleted = $index;
					$partiallyDeleted;
					$shouldDelete;
					Remove-Item -Path $deletePath -Recurse -Confirm:$false;
				}
			} catch {
				return $false
			}
		}
		return $true
	}
	function Show-Toast {
		Param
		(
			[Parameter(Mandatory=$true, Position=0)]
			[int] $xcopyErrorCode,
			[Parameter(Mandatory=$true, Position=1)]
			[string] $src,
			[Parameter(Mandatory=$true, Position=2)]
			[string] $dest,
			[Parameter(Mandatory=$true, Position=3)]
			[bool] $overwrite,
			[Parameter(Mandatory=$false, Position=4)]
			[bool] $renameOk,
			[Parameter(Mandatory=$false, Position=5)]
			[bool] $deleteOk,
			[Parameter(Mandatory=$false, Position=6)]
			[int] $partiallyDeleted,
			[Parameter(Mandatory=$false, Position=7)]
			[int] $shouldHaveDeleted,
			[Parameter(Mandatory=$true, Position=8)]
			[string] $appTitle,
			[Parameter(Mandatory=$true, Position=9)]
			[int] $toastExpirationInMinutes
		)
		$titleSuccess = 'Your scheduled backup was successful';
		$titleFailure = 'Your scheduled backup has failed';
		$contentSuccess = 'Your folder ' + $src + ' has been backed up to ' + $dest + '. ';
		$contentFailure = 'Your folder ' + $src + ' has not been backed up to ' + $dest + '. ';
		$toastTemplate = 'ToastText02';
		$xcopyError1 = 'No files were found to copy.';
		$xcopyError4 = 'There was not enough memory or disk space (Or the folder does not exist anymore).';
		$xcopyError5 = 'A disk write error occurred.';
		$renameFailure = 'the backup folder could not be renamed.';
		$deleteFailure = '' + $partiallyDeleted + ' out of ' + $shouldHaveDeleted + ' old backups have been removed.';
		$toastTitle = $null;
		$toastContent = $null;

		if ($xcopyErrorCode -EQ 0) {
			$toastTitle = [DateTime]::Now.ToShortTimeString() + ': ' + $titleSuccess;
			if(($renameOk -EQ $true) -AND ($deleteOk -EQ $true) -AND ($overwrite -EQ $true)) {
				$toastContent = $contentSuccess + 'There were no errors.';
			}
			if(($renameOk -EQ $true) -AND ($deleteOk -EQ $true) -AND ($overwrite -EQ $false)) {
				$toastContent = $contentSuccess + $shouldHaveDeleted + ' old backup(s) have been removed. There were no errors.';
			}
			if($renameOk -EQ $false) {
				$toastContent = $contentSuccess + 'However, ' + $renameFailure;
			}
			if($deleteOk -EQ $false) {
				$toastContent = $contentSuccess + 'However, ' + $deleteFailure;
			}
			if(($renameOk -EQ $false) -AND ($deleteOk -EQ $false)) {
				$toastContent = $contentSuccess + 'However, ' + $renameFailure + ' ' + $deleteFailure;
			}
		}
		if ($xcopyErrorCode -EQ 1) {
			$toastTitle = [DateTime]::Now.ToShortTimeString() + ': ' + $titleFailure; 
			$toastContent = $contentFailure + $xcopyError1;
		}
		if ($xcopyErrorCode -EQ 4) {
			$toastTitle = [DateTime]::Now.ToShortTimeString() + ': ' + $titleFailure; 
			$toastContent = $contentFailure + $xcopyError4;
		}
		if ($xcopyErrorCode -EQ 5) {
			$toastTitle = [DateTime]::Now.ToShortTimeString() + ': ' + $titleFailure;
			$toastContent = $contentFailure + $xcopyError5;
		}

		[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null; 
		$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::$toastTemplate); 
		$toastXml = [xml] $template.GetXml(); $toastXml.GetElementsByTagName('text')[0].AppendChild($toastXml.CreateTextNode($toastTitle)) > $null; 
		$toastXml.GetElementsByTagName('text')[1].AppendChild($toastXml.CreateTextNode($toastContent)) > $null; 
		$actionsElement = $toastXml.CreateElement('actions'); $actionElement = $toastXml.CreateElement('action'); 
		$actionElement.SetAttribute('content', 'Dismiss'); $actionElement.SetAttribute('arguments', 'dismiss'); 
		$actionElement.SetAttribute('activationType', 'system'); 
		$actionsElement.AppendChild($actionElement); 
		$toastXml.DocumentElement.AppendChild($actionsElement); 
		$xml = New-Object Windows.Data.Xml.Dom.XmlDocument; 
		$xml.LoadXml($toastXml.OuterXml); 
		$toast = [Windows.UI.Notifications.ToastNotification]::new($xml); 
		$toast.Tag = $appTitle; 
		$toast.Group = $appTitle; 
		$toast.ExpirationTime = [DateTimeOffset]::Now.AddMinutes($toastExpirationInMinutes); 
		if($actioncentre) { $toast.SuppressPopup = $true; };
		$notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($appTitle); 
		$notifier.Show($toast);
	}
	function Run-Backup {
		$ErrorActionPreference = 'Stop';
		$src = '%[1]v';
		$dest = '%[2]v';
		$folderName = '%[3]v';
		$destPath = $dest + '\' + $folderName;
		$overwrite = $%[6]v;
		$backupLimit = %[5]v;
		$appTitle = '%[4]v';
		$toastExpirationInMinutes = %[7]v;

		Copy-Folder $src $destPath;
		$xcopyErrorCode = $LASTEXITCODE;
		if (($overwrite -EQ $true) -OR ($xcopyErrorCode -NE 0)) {
			Show-Toast $xcopyErrorCode $src $dest $overwrite -renameOk $true -deleteOk $true -appTitle $apptitle -toastExpirationInMinutes $toastExpirationInMinutes;
			return
		}
		$renameOk = Rename-Backup $destPath $folderName
		if($backupLimit -EQ 0) {
			Show-Toast $xcopyErrorCode $src $dest $overwrite $renameOk $true -appTitle $apptitle -toastExpirationInMinutes $toastExpirationInMinutes;
			return
		}
		$deleted = Remove-Backup $backupLimit $dest $folderName;
		$deleteOk = $deleted[-1];
		$shouldHaveDeleted = $deleted[-2];
		$partiallyDeleted = $deleted[-3];

		Show-Toast $xcopyErrorCode $src $dest $overwrite $renameOk $deleteOk $partiallyDeleted $shouldHaveDeleted -appTitle $apptitle -toastExpirationInMinutes $toastExpirationInMinutes;
		return
	}
	Run-Backup
	`, src, dest, folder, appTitle, backupLimit, overwrite, toastExpirationTimeInMinutes)
}
