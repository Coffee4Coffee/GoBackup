# Script file for testing purposes

# Copy folder
function Copy-Folder($src, $dest) {
    xcopy $src $destPath /e /y /i /h /o /k /x;
    return
}

# Rename backup
function Rename-Backup($destPath, $folder) {
    try {
        $newFolderName = $folder + '-' + $(Get-Date -Format 'yyyyMMdd_HHmmss');
        Rename-Item  $destPath $newFolderName
    } catch {
        return $false
    }
    return $true
}

# Remove old backups
function Remove-Backup($backupLimit, $folderName) {
    $searchPattern = $folderName + '-' + '20[0-9][0-9][0-9][0-9][0-9][0-9]_[0-9][0-9][0-9][0-9][0-9][0-9]'
    # Sorted ascending by CreationTime, not LastWriteTime!
    # Since the folders have timestamps, we could also use those, but they can be changed by the user
    $backupFolderArray = Get-ChildItem -Directory $searchPattern | Sort-Object CreationTime

    # How many folders are over the limit? (This is normally only one folder)
    if(($backupFolderArray.length - $backupLimit) -GT 0) {
        try {
            $shouldDelete = $backupFolderArray.length - $backupLimit
            for ($index = 0; $index -lt $backupFolderArray.length - $backupLimit; $index++) {
                $partiallyDeleted = $index

                # Return a report on how many folder were deleted (will be returned as part of an array, in case of a partial delete)
                $partiallyDeleted
                $shouldDelete
                Remove-Item $backupFolderArray[$index].Name -Recurse -Confirm:$false
            }
        } catch {
            return $false
        }
    }
    return $true
}

# Show toast
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

    $titleSuccess = 'Your scheduled backup was successful'
    $titleFailure = 'Your scheduled backup has failed'
    $contentSuccess = 'Your folder ' + $src + ' has been backed up to ' + $dest + '. '
    $contentFailure = 'Your folder ' + $src + ' has not been backed up to ' + $dest + '. '
    $toastTemplate = 'ToastText02'

    $xcopyError1 = 'No files were found to copy.'
    $xcopyError4 = 'There was not enough memory or disk space (Or the folder does not exist anymore).'
    $xcopyError5 = 'A disk write error occurred.'

    $renameFailure = 'the backup folder could not be renamed.'
    $deleteFailure = '' + $partiallyDeleted + ' out of ' + $shouldHaveDeleted + ' old backups have been removed.'

    $toastTitle = $null
    $toastContent = $null

    if ($xcopyErrorCode -EQ 0) {
        $toastTitle = [DateTime]::Now.ToShortTimeString() + ': ' + $titleSuccess;
        # We can still have rename and delete errors, in that case the backup was partially a success
        if(($renameOk -EQ $true) -AND ($deleteOk -EQ $true) -AND ($overwrite -EQ $true)) {
            $toastContent = $contentSuccess + 'There were no errors.';
        }
        if(($renameOk -EQ $true) -AND ($deleteOk -EQ $true) -AND ($overwrite -EQ $false)) {
            $toastContent = $contentSuccess + $shouldHaveDeleted + ' old backup(s) have been removed. There were no errors.';
        }
        if($renameOk -EQ $false) {
            $toastContent = $contentSuccess + 'However, ' + $renameFailure
        }
        if($deleteOk -EQ $false) {
            $toastContent = $contentSuccess + 'However, ' + $deleteFailure
        }
        if(($renameOk -EQ $false) -AND ($deleteOk -EQ $false)) {
            $toastContent = $contentSuccess + 'However, ' + $renameFailure + ' ' + $deleteFailure
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

# main: copy (required), rename (optional), delete (optional)
function Run-Backup {
    # Terminate actions with an error, which would normally not throw one (e.g. Rename-Item)
    $ErrorActionPreference = 'Stop';
    $src = ''
    $dest = ''
    $folderName = ''
    $destPath = $dest + '\' + $folderName
    $overwrite = $false
    $backupLimit = 0
    $appTitle = '';
    $toastExpirationInMinutes = 0;

    # Execute backup using xcopy, if we overwrite or an error occured, exit here
    Copy-Folder $src $destPath
    $xcopyErrorCode = $LASTEXITCODE
    if (($overwrite -EQ $true) -OR ($xcopyErrorCode -NE 0)) {
        Show-Toast $xcopyErrorCode $src $dest $overwrite -renameOk $true -deleteOk $true  -appTitle $apptitle -toastExpirationInMinutes $toastExpirationInMinutes
        return
    }

    # Rename backup folder and add a timestamp
    $renameOk = Rename-Backup $destPath $folderName
    if($backupLimit -EQ 0) {
        Show-Toast $xcopyErrorCode $src $dest $overwrite $renameOk $true  -appTitle $apptitle -toastExpirationInMinutes $toastExpirationInMinutes
        return
    }
    # Rename was not possible, continue optionally with deleting old folders, keep this error and include it in the final toast report

    # $deleted is an array containing [..., $partiallyDeleted, $shouldDelete, $true/$false]
    $deleted = Remove-Backup $backupLimit $folderName
    # Last item in $deleted is $true (all deleted) or $false (none or partially deleted)
    $deleteOk = $deleted[-1]
    $shouldHaveDeleted = $deleted[-2]
    $partiallyDeleted = $deleted[-3]

    Show-Toast $xcopyErrorCode $src $dest $overwrite $renameOk $deleteOk $partiallyDeleted $shouldHaveDeleted  -appTitle $apptitle -toastExpirationInMinutes $toastExpirationInMinutes
    return
}



# Test paths
# overwrite
# copy: success +
# copy: failure +

# !overwrite
# copy: success, rename: success, delete: success +
# copy: success, rename: success, delete: failure +
# copy: success, rename: failure, delete: success +
# copy: success, rename: failure, delete: failure +