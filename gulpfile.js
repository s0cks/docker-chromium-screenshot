import gulp from 'gulp';
import * as dartSass from 'sass';
import gulpSass from 'gulp-sass';

const sassCompiler = gulpSass(dartSass);

// 1. Compile and compress Sass
export function styles() {
  return gulp.src('src/scss/**/*.scss')
    .pipe(sassCompiler({ outputStyle: 'compressed' }).on('error', sassCompiler.logError))
    .pipe(gulp.dest('dist/css'));
}

// 2. Copy public resources (images, fonts, html) as-is
export function publicFiles() {
  return gulp.src('public/**/*', { encoding: false }) // encoding: false preserves binary files like images
    .pipe(gulp.dest('dist'));
}

// 3. Main build task
export const build = gulp.parallel(styles, publicFiles);
