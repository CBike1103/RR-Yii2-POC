# RoadRunner + php7 + yii2 性能优化

## 工具
* RoadRunner: 可以作为nginx+fpm的替代品，用Go实现，亮点是用长时运行的PHP脚本作为worker.
* wrk: 轻量级http压测工具，可以用lua编写一些复杂场景测试.
* xhprof: PHP性能追踪分析工具
× yii2-psr7-bridge: psr7到yii2的bridge，同时提供了一些将yii2作为长时运行程序时的方法。

## 过程
### 1. 确认配置
php关于fpm和cli的设置有可能是分开的。fpm的设置可以在phpinfo()中查看，cli的设置在控制台查看。

确认php的opcache已安装，并且已经打开。确认php cli的配置中opcache.enable_cli已开启。

```
php -i | grep opcache
```

### 2. 创建一个yii2 demo项目

```
composer create-project --prefer-dist yiisoft/yii2-app-basic rrdemo
```

配置nginx指向该demo的web文件夹下的index.php，然后wrk跑一遍。
```
wrk -t 4 -c 100 http://localhost:xxxx
```

编写roadrunner程序，指向demo下的yii2-psr7-bridge脚本。
```php
<?php

ini_set('display_errors', 'stderr');
error_reporting(E_ALL);

defined('YII_DEBUG') or define('YII_DEBUG', true);
defined('YII_ENV') or define('YII_ENV', 'dev');

require __DIR__ . '/vendor/autoload.php';
require __DIR__ . '/vendor/yiisoft/yii2/Yii.php';

// Roadrunner relay and PSR7 object
$relay = new \Spiral\Goridge\StreamRelay(STDIN, STDOUT);
$psr7 = new \Spiral\RoadRunner\PSR7Client(new \Spiral\RoadRunner\Worker($relay));

$config = require __DIR__ . '/config/roadrunner.php';
$application = (new \yii\Psr7\web\Application($config));

// Handle each request in a loop
while ($request = $psr7->acceptRequest()) {
    // start profiling
    // xhprof_enable();
    try {
        $response = $application->handle($request);
        $psr7->respond($response);
    } catch (\Throwable $e) {
        // \yii\Psr7\web\ErrorHandler should handle any exceptions
        $psr7->getWorker()->error((string)$e);
        $psr7->getWorker()->stop();
    }
    if ($application->clean()) {
        $psr7->getWorker()->stop();
        return;
    }
}
```



```php
<?php

ini_set('display_errors', 'stderr');
error_reporting(E_ALL);

defined('YII_DEBUG') or define('YII_DEBUG', true);
defined('YII_ENV') or define('YII_ENV', 'prod');

require __DIR__ . '/vendor/autoload.php';
require __DIR__ . '/vendor/yiisoft/yii2/Yii.php';

// Roadrunner relay and PSR7 object
$relay = new \Spiral\Goridge\StreamRelay(STDIN, STDOUT);
$psr7 = new \Spiral\RoadRunner\PSR7Client(new \Spiral\RoadRunner\Worker($relay));

$config = require __DIR__ . '/config/roadrunner.php';
$application = (new \yii\Psr7\web\Application($config));

// // xhprof
// $XHPROF_ROOT = '/home/chenfang/Codes/general/xhprof';
// include_once $XHPROF_ROOT . "/xhprof_lib/utils/xhprof_lib.php";
// include_once $XHPROF_ROOT . "/xhprof_lib/utils/xhprof_runs.php";
// // save raw data for this profiler run using default
// // implementation of iXHProfRuns.
// $xhprof_runs = new XHProfRuns_Default();


// Handle each request in a loop
while ($request = $psr7->acceptRequest()) {
    // start profiling
    // xhprof_enable();
    try {
        $response = $application->handle($request);
        $psr7->respond($response);
    } catch (\Throwable $e) {
        // \yii\Psr7\web\ErrorHandler should handle any exceptions
        $psr7->getWorker()->error((string)$e);
        $psr7->getWorker()->stop();
    }

    // Workers will steadily grow in memory with each request until PHP memory_limit is reached, resulting in a worker crash.
    // With RoadRunner, you can tell the worker to shutdown if it approaches 10% of the maximum memory limit, allowing you to achieve better uptime.

    // unset($application);
    // gc_collect_cycles();
    if ($application->clean(20)) {
        $psr7->getWorker()->stop();
        return;
    }

    // $xhprof_data = xhprof_disable();
    // $run_id = $xhprof_runs->save_run($xhprof_data, "xhprof_support_roadrunner");
}
```