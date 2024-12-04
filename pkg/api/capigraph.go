package api

import (
	"fmt"
	"strings"

	"github.com/capillariesio/capillaries/pkg/capigraph"
	"github.com/capillariesio/capillaries/pkg/custom/py_calc"
	"github.com/capillariesio/capillaries/pkg/custom/tag_and_denormalize"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

const CapillariesIcons100x100 = `
<g id="icon-database-table-distinct">
  <g transform="scale(0.56) translate(2,61)">
    <path fill-rule="evenodd"
      d="M16.49,24.88C24.05,27.41,34.57,29,46.26,29S68.48,27.41,76,24.88c6.63-2.22,10.73-4.9,10.73-7.52S82.67,12.06,76,9.84C68.48,7.33,58,5.75,46.27,5.75S24.06,7.33,16.49,9.84c-14.06,4.7-14.46,10.21,0,15ZM64.91,55.34h48.73a9.27,9.27,0,0,1,9.24,9.24v42.58a9.27,9.27,0,0,1-9.24,9.25H64.91a9.27,9.27,0,0,1-9.24-9.25V64.58a9.27,9.27,0,0,1,9.24-9.24ZM91.09,99.18H118v12H91.09v-12Zm-30.89,0H87.13v12H60.2v-12Zm0-31.89H87.13v12H60.2v-12Zm0,15.94H87.13v12H60.2v-12ZM91.09,67.29H118v12H91.09v-12Zm0,15.94H118v12H91.09v-12ZM5.82,45.77c.52,2.45,4.5,4.91,10.68,7,7.22,2.42,17.16,3.95,28.24,4.08v5.77c-11.67-.13-22.25-1.78-30.05-4.39A35.86,35.86,0,0,1,5.84,54V71.27c.52,2.45,4.5,4.91,10.68,7,7.22,2.4,17.15,3.94,28.22,4.07v5.75c-11.67-.14-22.25-1.78-30.05-4.4A36.08,36.08,0,0,1,5.83,79.5V96.75c.52,2.45,4.51,4.91,10.68,7,7.22,2.41,17.16,4,28.23,4.08v5.75c-11.67-.13-22.24-1.78-30-4.4C10.4,107.72,0,103,0,97.38V95.55C0,69.86,0,43.06,0,17.41c0-5.43,5.61-10,14.66-13C22.82,1.68,34,0,46.27,0S69.7,1.68,77.87,4.41s13.64,6.78,14.55,11.53a3,3,0,0,1,.16,1v28.6H86.8V26.09a36.69,36.69,0,0,1-8.93,4.22c-8.15,2.75-19.31,4.41-31.58,4.41S22.83,33,14.66,30.31A36.26,36.26,0,0,1,5.8,26.14V45.77Z" />
  </g>
  <g transform="scale(0.9) translate(37,-25)">
    <path class="cls-1" d="m32.8,63.61h-6.75V28.44h6.75v35.17Z"/>
    <path class="cls-1" d="m40.45,28.44h17.35c4.85,0,7.91.95,9.91,3.01,2.53,2.58,3.38,5.59,3.38,11.71,0,9.76-.32,12.6-1.63,15.13-1.95,3.69-5.48,5.33-11.44,5.33h-17.56V28.44Zm16.24,29.37c3.22,0,5.01-.63,6.06-2.11,1.16-1.63,1.48-3.74,1.48-9.76s-.26-8.12-1.37-9.65c-1-1.42-2.64-2-5.75-2h-9.91v23.52h9.49Z"/>
  </g>
</g>
<g id="icon-database-table-copy">
  <g transform="scale(0.56) translate(2,61)">
    <path fill-rule="evenodd"
      d="M16.49,24.88C24.05,27.41,34.57,29,46.26,29S68.48,27.41,76,24.88c6.63-2.22,10.73-4.9,10.73-7.52S82.67,12.06,76,9.84C68.48,7.33,58,5.75,46.27,5.75S24.06,7.33,16.49,9.84c-14.06,4.7-14.46,10.21,0,15ZM64.91,55.34h48.73a9.27,9.27,0,0,1,9.24,9.24v42.58a9.27,9.27,0,0,1-9.24,9.25H64.91a9.27,9.27,0,0,1-9.24-9.25V64.58a9.27,9.27,0,0,1,9.24-9.24ZM91.09,99.18H118v12H91.09v-12Zm-30.89,0H87.13v12H60.2v-12Zm0-31.89H87.13v12H60.2v-12Zm0,15.94H87.13v12H60.2v-12ZM91.09,67.29H118v12H91.09v-12Zm0,15.94H118v12H91.09v-12ZM5.82,45.77c.52,2.45,4.5,4.91,10.68,7,7.22,2.42,17.16,3.95,28.24,4.08v5.77c-11.67-.13-22.25-1.78-30.05-4.39A35.86,35.86,0,0,1,5.84,54V71.27c.52,2.45,4.5,4.91,10.68,7,7.22,2.4,17.15,3.94,28.22,4.07v5.75c-11.67-.14-22.25-1.78-30.05-4.4A36.08,36.08,0,0,1,5.83,79.5V96.75c.52,2.45,4.51,4.91,10.68,7,7.22,2.41,17.16,4,28.23,4.08v5.75c-11.67-.13-22.24-1.78-30-4.4C10.4,107.72,0,103,0,97.38V95.55C0,69.86,0,43.06,0,17.41c0-5.43,5.61-10,14.66-13C22.82,1.68,34,0,46.27,0S69.7,1.68,77.87,4.41s13.64,6.78,14.55,11.53a3,3,0,0,1,.16,1v28.6H86.8V26.09a36.69,36.69,0,0,1-8.93,4.22c-8.15,2.75-19.31,4.41-31.58,4.41S22.83,33,14.66,30.31A36.26,36.26,0,0,1,5.8,26.14V45.77Z" />
  </g>
  <g transform="scale(0.35) translate(160,5)">
    <path fill-rule="evenodd"
      d="M16.49,24.88C24.05,27.41,34.57,29,46.26,29S68.48,27.41,76,24.88c6.63-2.22,10.73-4.9,10.73-7.52S82.67,12.06,76,9.84C68.48,7.33,58,5.75,46.27,5.75S24.06,7.33,16.49,9.84c-14.06,4.7-14.46,10.21,0,15ZM64.91,55.34h48.73a9.27,9.27,0,0,1,9.24,9.24v42.58a9.27,9.27,0,0,1-9.24,9.25H64.91a9.27,9.27,0,0,1-9.24-9.25V64.58a9.27,9.27,0,0,1,9.24-9.24ZM91.09,99.18H118v12H91.09v-12Zm-30.89,0H87.13v12H60.2v-12Zm0-31.89H87.13v12H60.2v-12Zm0,15.94H87.13v12H60.2v-12ZM91.09,67.29H118v12H91.09v-12Zm0,15.94H118v12H91.09v-12ZM5.82,45.77c.52,2.45,4.5,4.91,10.68,7,7.22,2.42,17.16,3.95,28.24,4.08v5.77c-11.67-.13-22.25-1.78-30.05-4.39A35.86,35.86,0,0,1,5.84,54V71.27c.52,2.45,4.5,4.91,10.68,7,7.22,2.4,17.15,3.94,28.22,4.07v5.75c-11.67-.14-22.25-1.78-30.05-4.4A36.08,36.08,0,0,1,5.83,79.5V96.75c.52,2.45,4.51,4.91,10.68,7,7.22,2.41,17.16,4,28.23,4.08v5.75c-11.67-.13-22.24-1.78-30-4.4C10.4,107.72,0,103,0,97.38V95.55C0,69.86,0,43.06,0,17.41c0-5.43,5.61-10,14.66-13C22.82,1.68,34,0,46.27,0S69.7,1.68,77.87,4.41s13.64,6.78,14.55,11.53a3,3,0,0,1,.16,1v28.6H86.8V26.09a36.69,36.69,0,0,1-8.93,4.22c-8.15,2.75-19.31,4.41-31.58,4.41S22.83,33,14.66,30.31A36.26,36.26,0,0,1,5.8,26.14V45.77Z" />
  </g>
</g>
<g id="icon-database-table-read">
  <g transform="scale(0.56) translate(2,61)">
    <path fill-rule="evenodd"
      d="M16.49,24.88C24.05,27.41,34.57,29,46.26,29S68.48,27.41,76,24.88c6.63-2.22,10.73-4.9,10.73-7.52S82.67,12.06,76,9.84C68.48,7.33,58,5.75,46.27,5.75S24.06,7.33,16.49,9.84c-14.06,4.7-14.46,10.21,0,15ZM64.91,55.34h48.73a9.27,9.27,0,0,1,9.24,9.24v42.58a9.27,9.27,0,0,1-9.24,9.25H64.91a9.27,9.27,0,0,1-9.24-9.25V64.58a9.27,9.27,0,0,1,9.24-9.24ZM91.09,99.18H118v12H91.09v-12Zm-30.89,0H87.13v12H60.2v-12Zm0-31.89H87.13v12H60.2v-12Zm0,15.94H87.13v12H60.2v-12ZM91.09,67.29H118v12H91.09v-12Zm0,15.94H118v12H91.09v-12ZM5.82,45.77c.52,2.45,4.5,4.91,10.68,7,7.22,2.42,17.16,3.95,28.24,4.08v5.77c-11.67-.13-22.25-1.78-30.05-4.39A35.86,35.86,0,0,1,5.84,54V71.27c.52,2.45,4.5,4.91,10.68,7,7.22,2.4,17.15,3.94,28.22,4.07v5.75c-11.67-.14-22.25-1.78-30.05-4.4A36.08,36.08,0,0,1,5.83,79.5V96.75c.52,2.45,4.51,4.91,10.68,7,7.22,2.41,17.16,4,28.23,4.08v5.75c-11.67-.13-22.24-1.78-30-4.4C10.4,107.72,0,103,0,97.38V95.55C0,69.86,0,43.06,0,17.41c0-5.43,5.61-10,14.66-13C22.82,1.68,34,0,46.27,0S69.7,1.68,77.87,4.41s13.64,6.78,14.55,11.53a3,3,0,0,1,.16,1v28.6H86.8V26.09a36.69,36.69,0,0,1-8.93,4.22c-8.15,2.75-19.31,4.41-31.58,4.41S22.83,33,14.66,30.31A36.26,36.26,0,0,1,5.8,26.14V45.77Z" />
  </g>
  <g transform="scale(0.1) translate(540,20)">
    <path fill-rule="nonzero"
      d="M117.91 0h201.68c3.93 0 7.44 1.83 9.72 4.67l114.28 123.67c2.21 2.37 3.27 5.4 3.27 8.41l.06 310c0 35.43-29.4 64.81-64.8 64.81H117.91c-35.57 0-64.81-29.24-64.81-64.81V64.8C53.1 29.13 82.23 0 117.91 0zM325.5 37.15v52.94c2.4 31.34 23.57 42.99 52.93 43.5l36.16-.04-89.09-96.4zm96.5 121.3l-43.77-.04c-42.59-.68-74.12-21.97-77.54-66.54l-.09-66.95H117.91c-21.93 0-39.89 17.96-39.89 39.88v381.95c0 21.82 18.07 39.89 39.89 39.89h264.21c21.71 0 39.88-18.15 39.88-39.89v-288.3z" />
  </g>
</g>
<g id="icon-database-table-join">
  <g transform="scale(0.56) translate(2,61)">
    <path fill-rule="evenodd"
      d="M16.49,24.88C24.05,27.41,34.57,29,46.26,29S68.48,27.41,76,24.88c6.63-2.22,10.73-4.9,10.73-7.52S82.67,12.06,76,9.84C68.48,7.33,58,5.75,46.27,5.75S24.06,7.33,16.49,9.84c-14.06,4.7-14.46,10.21,0,15ZM64.91,55.34h48.73a9.27,9.27,0,0,1,9.24,9.24v42.58a9.27,9.27,0,0,1-9.24,9.25H64.91a9.27,9.27,0,0,1-9.24-9.25V64.58a9.27,9.27,0,0,1,9.24-9.24ZM91.09,99.18H118v12H91.09v-12Zm-30.89,0H87.13v12H60.2v-12Zm0-31.89H87.13v12H60.2v-12Zm0,15.94H87.13v12H60.2v-12ZM91.09,67.29H118v12H91.09v-12Zm0,15.94H118v12H91.09v-12ZM5.82,45.77c.52,2.45,4.5,4.91,10.68,7,7.22,2.42,17.16,3.95,28.24,4.08v5.77c-11.67-.13-22.25-1.78-30.05-4.39A35.86,35.86,0,0,1,5.84,54V71.27c.52,2.45,4.5,4.91,10.68,7,7.22,2.4,17.15,3.94,28.22,4.07v5.75c-11.67-.14-22.25-1.78-30.05-4.4A36.08,36.08,0,0,1,5.83,79.5V96.75c.52,2.45,4.51,4.91,10.68,7,7.22,2.41,17.16,4,28.23,4.08v5.75c-11.67-.13-22.24-1.78-30-4.4C10.4,107.72,0,103,0,97.38V95.55C0,69.86,0,43.06,0,17.41c0-5.43,5.61-10,14.66-13C22.82,1.68,34,0,46.27,0S69.7,1.68,77.87,4.41s13.64,6.78,14.55,11.53a3,3,0,0,1,.16,1v28.6H86.8V26.09a36.69,36.69,0,0,1-8.93,4.22c-8.15,2.75-19.31,4.41-31.58,4.41S22.83,33,14.66,30.31A36.26,36.26,0,0,1,5.8,26.14V45.77Z" />
  </g>
  <g transform="scale(0.1) translate(500,50)">
    <path fill-rule="nonzero"
      d="M303.633 363.721c10.832-11.26 28.745-11.608 40.007-.776 11.26 10.832 11.608 28.745.776 40.006l-91.965 95.954c-5.07 7.72-13.8 12.824-23.727 12.824-1.387 0-2.75-.1-4.079-.296l-.239-.033a28.149 28.149 0 01-15.859-7.649l-96.64-100.8c-10.832-11.261-10.484-29.174.777-40.006s29.174-10.484 40.006.776l47.665 49.733V258.99c0-50.724-20.558-101.577-53.822-139.863-31.152-35.856-73.279-60.35-119.571-62.576C11.355 55.817-.702 42.569.032 26.962.766 11.355 14.014-.702 29.621.032c62.738 3.021 118.895 35.136 159.687 82.086 15.498 17.837 28.798 37.876 39.416 59.302 10.579-21.35 23.828-41.327 39.253-59.121C308.656 35.368 364.703 3.22 427.379.051c15.607-.734 28.855 11.323 29.589 26.93.734 15.607-11.323 28.855-26.93 29.589-46.168 2.335-88.19 26.868-119.285 62.738-33.168 38.253-53.66 89.029-53.66 139.682v153.292l46.54-48.561z" />
  </g>
</g>
<g id="icon-database-table-py">
  <g transform="scale(0.56) translate(2,61)">
    <path fill-rule="evenodd"
      d="M16.49,24.88C24.05,27.41,34.57,29,46.26,29S68.48,27.41,76,24.88c6.63-2.22,10.73-4.9,10.73-7.52S82.67,12.06,76,9.84C68.48,7.33,58,5.75,46.27,5.75S24.06,7.33,16.49,9.84c-14.06,4.7-14.46,10.21,0,15ZM64.91,55.34h48.73a9.27,9.27,0,0,1,9.24,9.24v42.58a9.27,9.27,0,0,1-9.24,9.25H64.91a9.27,9.27,0,0,1-9.24-9.25V64.58a9.27,9.27,0,0,1,9.24-9.24ZM91.09,99.18H118v12H91.09v-12Zm-30.89,0H87.13v12H60.2v-12Zm0-31.89H87.13v12H60.2v-12Zm0,15.94H87.13v12H60.2v-12ZM91.09,67.29H118v12H91.09v-12Zm0,15.94H118v12H91.09v-12ZM5.82,45.77c.52,2.45,4.5,4.91,10.68,7,7.22,2.42,17.16,3.95,28.24,4.08v5.77c-11.67-.13-22.25-1.78-30.05-4.39A35.86,35.86,0,0,1,5.84,54V71.27c.52,2.45,4.5,4.91,10.68,7,7.22,2.4,17.15,3.94,28.22,4.07v5.75c-11.67-.14-22.25-1.78-30.05-4.4A36.08,36.08,0,0,1,5.83,79.5V96.75c.52,2.45,4.51,4.91,10.68,7,7.22,2.41,17.16,4,28.23,4.08v5.75c-11.67-.13-22.24-1.78-30-4.4C10.4,107.72,0,103,0,97.38V95.55C0,69.86,0,43.06,0,17.41c0-5.43,5.61-10,14.66-13C22.82,1.68,34,0,46.27,0S69.7,1.68,77.87,4.41s13.64,6.78,14.55,11.53a3,3,0,0,1,.16,1v28.6H86.8V26.09a36.69,36.69,0,0,1-8.93,4.22c-8.15,2.75-19.31,4.41-31.58,4.41S22.83,33,14.66,30.31A36.26,36.26,0,0,1,5.8,26.14V45.77Z" />
  </g>
  <g transform="scale(2.1) translate(24,0)">
    <path d="m9.8594 2.0009c-1.58 0-2.8594 1.2794-2.8594 2.8594v1.6797h4.2891c.39 0 .71094.57094.71094.96094h-7.1406c-1.58 0-2.8594 1.2794-2.8594 2.8594v3.7812c0 1.58 1.2794 2.8594 2.8594 2.8594h1.1797v-2.6797c0-1.58 1.2716-2.8594 2.8516-2.8594h5.25c1.58 0 2.8594-1.2716 2.8594-2.8516v-3.75c0-1.58-1.2794-2.8594-2.8594-2.8594zm-.71875 1.6094c.4 0 .71875.12094.71875.71094s-.31875.89062-.71875.89062c-.39 0-.71094-.30062-.71094-.89062s.32094-.71094.71094-.71094z"/>
    <path d="m17.959 7v2.6797c0 1.58-1.2696 2.8594-2.8496 2.8594h-5.25c-1.58 0-2.8594 1.2696-2.8594 2.8496v3.75a2.86 2.86 0 0 0 2.8594 2.8613h4.2812a2.86 2.86 0 0 0 2.8594 -2.8613v-1.6797h-4.291c-.39 0-.70898-.56898-.70898-.95898h7.1406a2.86 2.86 0 0 0 2.8594 -2.8613v-3.7793a2.86 2.86 0 0 0 -2.8594 -2.8594zm-9.6387 4.5137-.0039.0039c.01198-.0024.02507-.0016.03711-.0039zm6.5391 7.2754c.39 0 .71094.30062.71094.89062a.71 .71 0 0 1 -.71094 .70898c-.4 0-.71875-.11898-.71875-.70898s.31875-.89062.71875-.89062z"/>
  </g>
</g>
<g id="icon-database-table-tag">
  <g transform="scale(0.56) translate(2,61)">
    <path fill-rule="evenodd"
      d="M16.49,24.88C24.05,27.41,34.57,29,46.26,29S68.48,27.41,76,24.88c6.63-2.22,10.73-4.9,10.73-7.52S82.67,12.06,76,9.84C68.48,7.33,58,5.75,46.27,5.75S24.06,7.33,16.49,9.84c-14.06,4.7-14.46,10.21,0,15ZM64.91,55.34h48.73a9.27,9.27,0,0,1,9.24,9.24v42.58a9.27,9.27,0,0,1-9.24,9.25H64.91a9.27,9.27,0,0,1-9.24-9.25V64.58a9.27,9.27,0,0,1,9.24-9.24ZM91.09,99.18H118v12H91.09v-12Zm-30.89,0H87.13v12H60.2v-12Zm0-31.89H87.13v12H60.2v-12Zm0,15.94H87.13v12H60.2v-12ZM91.09,67.29H118v12H91.09v-12Zm0,15.94H118v12H91.09v-12ZM5.82,45.77c.52,2.45,4.5,4.91,10.68,7,7.22,2.42,17.16,3.95,28.24,4.08v5.77c-11.67-.13-22.25-1.78-30.05-4.39A35.86,35.86,0,0,1,5.84,54V71.27c.52,2.45,4.5,4.91,10.68,7,7.22,2.4,17.15,3.94,28.22,4.07v5.75c-11.67-.14-22.25-1.78-30.05-4.4A36.08,36.08,0,0,1,5.83,79.5V96.75c.52,2.45,4.51,4.91,10.68,7,7.22,2.41,17.16,4,28.23,4.08v5.75c-11.67-.13-22.24-1.78-30-4.4C10.4,107.72,0,103,0,97.38V95.55C0,69.86,0,43.06,0,17.41c0-5.43,5.61-10,14.66-13C22.82,1.68,34,0,46.27,0S69.7,1.68,77.87,4.41s13.64,6.78,14.55,11.53a3,3,0,0,1,.16,1v28.6H86.8V26.09a36.69,36.69,0,0,1-8.93,4.22c-8.15,2.75-19.31,4.41-31.58,4.41S22.83,33,14.66,30.31A36.26,36.26,0,0,1,5.8,26.14V45.77Z" />
  </g>
  <g transform="scale(0.25) translate(260,15)">
    <path fill-rule="evenodd" clip-rule="evenodd"
      d="M64.534,2.334l7.463,0.472c0.513-0.035,1.034-0.063,1.567-0.082L120.7,54.051 c3.183,3.465,2.819,8.945-0.69,12.076l-41.5,37c-3.037,2.709-7.527,2.838-10.707,0.541l42.466-37.861 c3.74-3.336,4.128-9.18,0.736-12.871L64.534,2.334L64.534,2.334z M53.067,3.185l45.534,49.581 c3.074,3.348,2.723,8.643-0.667,11.666l-40.089,35.744c-3.387,3.02-8.645,2.719-11.666-0.67L3.422,51.541l0.082-1.414L0,0 l51.549,3.264C52.045,3.23,52.551,3.203,53.067,3.185L53.067,3.185z M20.058,9.77c5.104,0.112,9.154,4.345,9.042,9.45 s-4.345,9.155-9.45,9.042c-5.105-0.112-9.154-4.345-9.042-9.45C10.72,13.707,14.953,9.657,20.058,9.77L20.058,9.77z M28.251,44.743 l29.426,32.062l-5.865,5.385L22.386,50.125L28.251,44.743L28.251,44.743z M41.265,33.13l29.429,32.061l-5.867,5.383L35.399,38.513 L41.265,33.13L41.265,33.13z M53.091,22.061L82.52,54.125l-5.867,5.383L47.226,27.447L53.091,22.061L53.091,22.061z" />
  </g>
  <g transform="scale(0.28) translate(220,120)">
    <path fill-rule="evenodd" clip-rule="evenodd" stroke="#000000" stroke-width="0.5"
      d="M23.838,62.429 L0.352,85.799l23.486,24.071V94.695c15.025,0.074,26.659-4.704,34.892-14.394c1.013-1.192,1.965-2.454,2.855-3.786 c0.995,1.562,2.069,3.023,3.223,4.382c8.24,9.698,19.891,14.477,34.936,14.393v15.175l23.486-24.07l-23.486-23.37v13.3 c-9.197,0.059-16.012-2.497-20.432-7.699c-4.93-5.8-7.615-15.109-8.055-27.966V0.25H52.281v40.4 c-0.438,12.861-3.125,21.578-8.054,27.38c-4.414,5.193-11.213,7.749-20.389,7.698V62.429L23.838,62.429z" />
  </g>
</g>
<g id="icon-database-table" transform="scale(0.8,0.8) translate(139,4)">
  <path fill-rule="evenodd"
    d="M16.49,24.88C24.05,27.41,34.57,29,46.26,29S68.48,27.41,76,24.88c6.63-2.22,10.73-4.9,10.73-7.52S82.67,12.06,76,9.84C68.48,7.33,58,5.75,46.27,5.75S24.06,7.33,16.49,9.84c-14.06,4.7-14.46,10.21,0,15ZM64.91,55.34h48.73a9.27,9.27,0,0,1,9.24,9.24v42.58a9.27,9.27,0,0,1-9.24,9.25H64.91a9.27,9.27,0,0,1-9.24-9.25V64.58a9.27,9.27,0,0,1,9.24-9.24ZM91.09,99.18H118v12H91.09v-12Zm-30.89,0H87.13v12H60.2v-12Zm0-31.89H87.13v12H60.2v-12Zm0,15.94H87.13v12H60.2v-12ZM91.09,67.29H118v12H91.09v-12Zm0,15.94H118v12H91.09v-12ZM5.82,45.77c.52,2.45,4.5,4.91,10.68,7,7.22,2.42,17.16,3.95,28.24,4.08v5.77c-11.67-.13-22.25-1.78-30.05-4.39A35.86,35.86,0,0,1,5.84,54V71.27c.52,2.45,4.5,4.91,10.68,7,7.22,2.4,17.15,3.94,28.22,4.07v5.75c-11.67-.14-22.25-1.78-30.05-4.4A36.08,36.08,0,0,1,5.83,79.5V96.75c.52,2.45,4.51,4.91,10.68,7,7.22,2.41,17.16,4,28.23,4.08v5.75c-11.67-.13-22.24-1.78-30-4.4C10.4,107.72,0,103,0,97.38V95.55C0,69.86,0,43.06,0,17.41c0-5.43,5.61-10,14.66-13C22.82,1.68,34,0,46.27,0S69.7,1.68,77.87,4.41s13.64,6.78,14.55,11.53a3,3,0,0,1,.16,1v28.6H86.8V26.09a36.69,36.69,0,0,1-8.93,4.22c-8.15,2.75-19.31,4.41-31.58,4.41S22.83,33,14.66,30.31A36.26,36.26,0,0,1,5.8,26.14V45.77Z" />
</g>
<g id="icon-csv">
  <g transform="scale(0.193,0.193) translate(11,3)">
    <path fill-rule="nonzero"
      d="M117.91 0h201.68c3.93 0 7.44 1.83 9.72 4.67l114.28 123.67c2.21 2.37 3.27 5.4 3.27 8.41l.06 310c0 35.43-29.4 64.81-64.8 64.81H117.91c-35.57 0-64.81-29.24-64.81-64.81V64.8C53.1 29.13 82.23 0 117.91 0zM325.5 37.15v52.94c2.4 31.34 23.57 42.99 52.93 43.5l36.16-.04-89.09-96.4zm96.5 121.3l-43.77-.04c-42.59-.68-74.12-21.97-77.54-66.54l-.09-66.95H117.91c-21.93 0-39.89 17.96-39.89 39.88v381.95c0 21.82 18.07 39.89 39.89 39.89h264.21c21.71 0 39.88-18.15 39.88-39.89v-288.3z" />
  </g>
  <g transform="scale(0.25) translate(12,60)">
    <path fill-rule="nonzero"
      d="M133.97 242.65l1.98 22.2c-5.54 2.29-12.5 3.44-20.87 3.44-8.37 0-15.09-.88-20.16-2.65-5.06-1.76-9.05-4.53-11.95-8.32-2.91-3.79-4.94-8.24-6.08-13.35-1.15-5.11-1.72-11.4-1.72-18.89s.57-13.81 1.72-18.96c1.14-5.16 3.17-9.63 6.08-13.42 5.63-7.31 15.99-10.96 31.05-10.96 3.35 0 7.29.33 11.82.99 4.54.66 7.91 1.47 10.11 2.44l-3.96 20.22c-5.73-1.23-10.97-1.85-15.72-1.85-4.76 0-8.07.44-9.92 1.32-1.84.88-2.77 2.64-2.77 5.29v34.62c3.43.7 6.91 1.05 10.44 1.05 7.49 0 14.14-1.05 19.95-3.17zm11.9 22.2l3.69-21.8c8.11 2.03 15.4 3.04 21.87 3.04 6.48 0 11.7-.27 15.66-.79v-6.61l-11.89-1.06c-10.75-.97-18.13-3.54-22.13-7.73-4.01-4.18-6.02-10.37-6.02-18.56 0-11.28 2.45-19.03 7.34-23.26s13.19-6.34 24.91-6.34 22.28 1.1 31.71 3.3l-3.3 21.14c-8.19-1.32-14.76-1.98-19.69-1.98-4.93 0-9.12.22-12.56.66v6.48l9.52.92c11.54 1.15 19.51 3.9 23.92 8.26 4.4 4.36 6.6 10.42 6.6 18.17 0 5.55-.74 10.24-2.24 14.07-1.5 3.83-3.28 6.74-5.35 8.72-2.07 1.99-5 3.5-8.79 4.56-3.79 1.06-7.11 1.7-9.98 1.92-2.86.22-6.67.33-11.43.33-11.45 0-22.07-1.15-31.84-3.44zm128.57-81.13h27.88l-20.48 82.59h-38.59l-20.48-82.59h27.88l11.24 52.46h1.19l11.36-52.46z" />
  </g>
</g>
<g id="icon-parquet" transform="scale(0.193,0.193) translate(11,3)">
  <path fill-rule="nonzero"
    d="M117.91 0h201.68c3.93 0 7.44 1.83 9.72 4.67l114.28 123.67c2.21 2.37 3.27 5.4 3.27 8.41l.06 310c0 35.43-29.4 64.81-64.8 64.81H117.91c-35.57 0-64.81-29.24-64.81-64.81V64.8C53.1 29.13 82.23 0 117.91 0zM325.5 37.15v52.94c2.4 31.34 23.57 42.99 52.93 43.5l36.16-.04-89.09-96.4zm96.5 121.3l-43.77-.04c-42.59-.68-74.12-21.97-77.54-66.54l-.09-66.95H117.91c-21.93 0-39.89 17.96-39.89 39.88v381.95c0 21.82 18.07 39.89 39.89 39.89h264.21c21.71 0 39.88-18.15 39.88-39.89v-288.3z" />
  <g transform="scale(5,5) translate(18,45)">
    <path class="b"
      d="M29,47.3c-2.5-2.7-2.6-2.8-2.1-3.3,.7-.8,31.4-28.9,31.8-29.1,.4-.2,4.1,1.6,5,2.4,.3,.3-2.1,2.9-15.5,16.5-8.7,8.9-16,16.2-16.2,16.2-.2,0-1.5-1.2-3-2.7h0Z" />
    <path class="b"
      d="M22.1,40.1c-1.4-1.4-2.2-2.4-2.1-2.7,.2-.4,17.8-14.7,18.1-14.6,.4,0,4.6,3.2,4.6,3.5s-17.7,16.1-18.1,16.1-1.2-1-2.5-2.3h0Z" />
    <path class="b"
      d="M15.7,33.6c-1.1-1.2-1.7-2.2-1.6-2.3,.7-.7,8.4-6,8.7-6,.4,0,4.1,3.2,4.1,3.5s-8.3,6.8-8.9,6.8-1.3-.9-2.3-2.1Z" />
    <path class="b"
      d="M10.4,28.2c-1.5-1.5-1.7-1.8-1.4-2.2,.5-.5,24.9-16.2,25.4-16.4,.4-.1,3.8,1.9,3.8,2.3S13.2,29.7,12.6,29.8c-.2,0-1.2-.6-2.2-1.7h0Z" />
    <path class="b"
      d="M26.7,25.7c-1-.8-1.8-1.7-1.8-1.8,0-.2,.9-.9,2-1.7,1.1-.8,5.9-4.2,10.8-7.7,4.8-3.5,9-6.3,9.2-6.3,.7,0,3.9,1.9,3.8,2.2-.2,.5-21.3,16.8-21.7,16.8-.2,0-1.2-.7-2.2-1.5h0Z" />
    <path class="b"
      d="M5.5,23.2c-1.2-1.3-1.4-1.6-1.1-1.9,.3-.2,15.4-9.2,17.2-10.2,.3-.2,3.5,2.1,3.4,2.5-.1,.4-17.2,11.1-17.7,11.1-.2,0-1.1-.7-1.9-1.5h0Z" />
    <path class="b"
      d="M42,23.3c-1.1-.8-2.1-1.6-2.1-1.7,0-.2,9.9-8.5,12.2-10.1,.4-.3,.8-.2,2.7,.8,1.2,.6,2.2,1.3,2.2,1.4,0,.4-11.9,10.9-12.5,11-.3,0-1.4-.6-2.5-1.4h0Z" />
    <path class="b"
      d="M1.5,18.9l-1.5-1.5,.6-.4c.4-.2,7.5-4.2,16-8.7L32,0l1.5,.7c.8,.4,1.5,.9,1.5,1S3.4,20.4,3,20.4s-.7-.7-1.5-1.5h0Z" />
    <path class="b"
      d="M25,11.5c-.7-.5-1.3-1.1-1.3-1.3,0-.2,2.9-2,6.4-4.1l6.4-3.8,1.6,.9c.9,.5,1.6,1,1.6,1.1,0,.3-12.5,8.2-13,8.1-.2,0-1-.4-1.7-1h0Z" />
    <path class="b"
      d="M37.8,9.9c-.9-.5-1.6-1.1-1.6-1.3,0-.3,4.4-3.2,5.1-3.4,.3,0,3.3,1.5,3.5,1.8,0,.1,0,.4-.3,.6-.9,.8-4.5,3.2-4.8,3.2-.2,0-1-.4-1.9-1h0Z" />
  </g>
</g>`

func NodeBatchStatusToCapigraphColor(status wfmodel.NodeBatchStatusType) int32 {
	switch status {
	case wfmodel.NodeBatchNone:
		return 0 // not colored, white
	case wfmodel.NodeBatchStart:
		return 0x0000FF // blue
	case wfmodel.NodeBatchSuccess:
		return 0x008000 // darkgreen
	case wfmodel.NodeBatchFail:
		return 0xFF0000 // red
	case wfmodel.NodeBatchRunStopReceived:
		return 0xFF8C00 // darkorange
	default:
		return 0x2F4F4F // darkslategray
	}
}

func shortenFileName(s string) string {
	parts := strings.Split(s, "/")
	sb := strings.Builder{}
	for i, s := range parts {
		if i < 3 || i == len(parts)-1 {
			if i > 0 {
				sb.WriteString("/")
			}
			sb.WriteString(s)
		} else if i == 3 {
			sb.WriteString("/...")
		}
	}
	return sb.String()
}

func nodeTypeDescription(node *sc.ScriptNodeDef) string {
	switch node.Type {
	case sc.NodeTypeFileTable:
		return fmt.Sprintf(
			"Processor: read from files into a table\n"+
				"File(s): %s\n"+
				"Table created: %s",
			shortenFileName(node.FileReader.SrcFileUrls[0]),
			node.TableCreator.Name)
	case sc.NodeTypeTableTable:
		return fmt.Sprintf(
			"Processor: write from table to table\n"+
				"Source table: %s\n"+
				"Table created: %s",
			node.TableReader.TableName,
			node.TableCreator.Name)
	case sc.NodeTypeTableLookupTable:
		return fmt.Sprintf(
			"Processor: %s join with lookup table, group: %t\n"+
				"Lookup index: %s\n"+
				"Table created: %s",
			node.Lookup.LookupJoin,
			node.Lookup.IsGroup,
			node.Lookup.IndexName,
			node.TableCreator.Name)
	case sc.NodeTypeTableFile:
		return fmt.Sprintf(
			"Processor: read from table into files\n"+
				"Source table: %s\n"+
				"File(s): %s",
			node.TableReader.TableName,
			shortenFileName(node.FileCreator.UrlTemplate))
	case sc.NodeTypeTableCustomTfmTable:
		switch node.CustomProcessorType {
		case py_calc.ProcessorPyCalcName:
			return fmt.Sprintf(
				"Processor: apply Python calculations\n"+
					"Python file(s): %s\n"+
					"Table created: %s",
				shortenFileName(node.CustomProcessor.(*py_calc.PyCalcProcessorDef).PythonUrls[0]),
				node.TableCreator.Name)
		case tag_and_denormalize.ProcessorTagAndDenormalizeName:
			tagCriteriaUrl := shortenFileName(node.CustomProcessor.(*tag_and_denormalize.TagAndDenormalizeProcessorDef).RawTagCriteriaUri)
			if len(tagCriteriaUrl) == 0 {
				tagCriteriaUrl = "(inline)"
			}
			return fmt.Sprintf(
				"Processor: tag and denormalize\n"+
					"Tag criteria URL: %s\n"+
					"Table created: %s",
				tagCriteriaUrl,
				node.TableCreator.Name)
		default:
			return "Custom processor: unknown"
		}
	case sc.NodeTypeDistinctTable:
		distinctIdxName, _, err := node.TableCreator.GetSingleUniqueIndexDef()
		if err != nil {
			distinctIdxName = "unknown"
		}

		return fmt.Sprintf(
			"Processor: select distinct rows\n"+
				"Unique index: %s\n"+
				"Table created: %s",
			distinctIdxName,
			node.TableCreator.Name)
	default:
		return "Unknown processor type: " + string(node.Type)
	}
}

func nodeTypeIcon(node *sc.ScriptNodeDef) string {
	switch node.Type {
	case sc.NodeTypeFileTable:
		return "icon-database-table-read"
	case sc.NodeTypeTableTable:
		return "icon-database-table-copy"
	case sc.NodeTypeTableLookupTable:
		return "icon-database-table-join"
	case sc.NodeTypeTableFile:
		if node.FileCreator.CreatorFileType == sc.CreatorFileTypeCsv {
			return "icon-csv"
		} else if node.FileCreator.CreatorFileType == sc.CreatorFileTypeParquet {
			return "icon-parquet"
		} else {
			return ""
		}
	case sc.NodeTypeTableCustomTfmTable:
		switch node.CustomProcessorType {
		case py_calc.ProcessorPyCalcName:
			return "icon-database-table-py"
		case tag_and_denormalize.ProcessorTagAndDenormalizeName:
			return "icon-database-table-tag"
		default:
			return ""
		}
	case sc.NodeTypeDistinctTable:
		return "icon-database-table-distinct"
	default:

		return ""
	}
}

func GetCapigraphDiagram(scriptDef *sc.ScriptDef, dotDiagramType DiagramType, nodeStringColorMap map[string]int32) string {
	nodeDefs := make([]capigraph.NodeDef, len(scriptDef.ScriptNodes)+1)
	nodeDefs[0] = capigraph.NodeDef{0, "", capigraph.EdgeDef{}, []capigraph.EdgeDef{}, "", 0, false}
	nodeNameMap := map[string]int16{}

	// Populate nodes
	nodeIdx := int16(1)
	for _, node := range scriptDef.ScriptNodes {
		nodeNameMap[node.Name] = nodeIdx
		color := int32(0)
		if nodeStringColorMap != nil {
			color = nodeStringColorMap[node.Name]
		}
		nodeDefs[nodeIdx] = capigraph.NodeDef{nodeIdx, fmt.Sprintf("%s\n%s\n%s", node.Name, node.Desc, nodeTypeDescription(node)), capigraph.EdgeDef{}, []capigraph.EdgeDef{}, nodeTypeIcon(node), color, node.StartPolicy == sc.NodeStartManual}
		nodeIdx++
	}

	// Populate direct parents and lookups
	for _, node := range scriptDef.ScriptNodes {
		nodeIdx := nodeNameMap[node.Name]
		allUsedFields := sc.FieldRefs{}
		if node.Type == sc.NodeTypeTableCustomTfmTable && node.CustomProcessorType == py_calc.ProcessorPyCalcName {
			usedInPyExpressions := node.CustomProcessor.(*py_calc.PyCalcProcessorDef).GetUsedInTargetExpressionsFields()
			allUsedFields.Append(*usedInPyExpressions)
		} else if node.Type == sc.NodeTypeTableCustomTfmTable && node.CustomProcessorType == tag_and_denormalize.ProcessorTagAndDenormalizeName {
			usedInTagExpressions := node.CustomProcessor.(*tag_and_denormalize.TagAndDenormalizeProcessorDef).GetUsedInTargetExpressionsFields()
			allUsedFields.Append(*usedInTagExpressions)
		}
		if node.HasFileCreator() {
			usedInAllTargetFileExpressions := node.FileCreator.GetFieldRefsUsedInAllTargetFileExpressions()
			allUsedFields.Append(usedInAllTargetFileExpressions)
		} else if node.HasTableCreator() {
			usedInAllTargetTableExpressions := sc.GetFieldRefsUsedInAllTargetExpressions(node.TableCreator.Fields)
			allUsedFields.Append(usedInAllTargetTableExpressions)
		}

		if node.HasTableReader() {
			parentNode := scriptDef.TableCreatorNodeMap[node.TableReader.TableName]
			parentNodeIdx := nodeNameMap[parentNode.Name]
			nodeDefs[nodeIdx].PriIn.SrcId = parentNodeIdx
			if dotDiagramType == DiagramType(DiagramIndexes) || dotDiagramType == DiagramType(DiagramRunStatus) {
				if node.TableReader.ExpectedBatchesTotal > 1 {
					nodeDefs[nodeIdx].PriIn.Text = fmt.Sprintf("%s\n(%d batches)", node.TableReader.TableName, node.TableReader.ExpectedBatchesTotal)
				} else {
					nodeDefs[nodeIdx].PriIn.Text = fmt.Sprintf("%s\n(no parallelism)", node.TableReader.TableName)
				}
			} else if dotDiagramType == DiagramType(DiagramFields) {
				sb := strings.Builder{}
				for i := 0; i < len(allUsedFields); i++ {
					if allUsedFields[i].TableName == sc.ReaderAlias {
						if sb.Len() > 0 {
							sb.WriteString("\n")
						}
						sb.WriteString(allUsedFields[i].FieldName)
					}
				}
				nodeDefs[nodeIdx].PriIn.Text = sb.String()
			}
		}
		if node.HasLookup() {
			lkpParentNode := scriptDef.IndexNodeMap[node.Lookup.IndexName]
			lkpParentNodeIdx := nodeNameMap[lkpParentNode.Name]

			inLkpArrowLabel := fmt.Sprintf("%s (lookup)", node.Lookup.IndexName)
			if dotDiagramType == DiagramType(DiagramFields) {
				inLkpArrowLabelBuilder := strings.Builder{}
				for i := 0; i < len(allUsedFields); i++ {
					if allUsedFields[i].TableName == sc.LookupAlias {
						if inLkpArrowLabelBuilder.Len() > 0 {
							inLkpArrowLabelBuilder.WriteString("\n")
						}
						inLkpArrowLabelBuilder.WriteString(allUsedFields[i].FieldName)

					}
				}
				inLkpArrowLabel = inLkpArrowLabelBuilder.String()
			}
			nodeDefs[nodeIdx].SecIn = append(nodeDefs[nodeIdx].SecIn, capigraph.EdgeDef{lkpParentNodeIdx, inLkpArrowLabel})
		}
	}

	svg, _, _, _, _, errOpt := capigraph.DrawOptimized(nodeDefs, capigraph.DefaultNodeFontOptions(), capigraph.DefaultEdgeLabelFontOptions(), capigraph.DefaultEdgeOptions(), CapillariesIcons100x100, "", capigraph.DefaultPalette())
	if errOpt != nil {
		var errUnopt error
		svg, _, errUnopt = capigraph.DrawUnoptimized(nodeDefs, capigraph.DefaultNodeFontOptions(), capigraph.DefaultEdgeLabelFontOptions(), capigraph.DefaultEdgeOptions(), CapillariesIcons100x100, "", capigraph.DefaultPalette())
		if errUnopt != nil {
			svg = fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 500 200">
<style>{font-family:arial; font-weight:normal; font-size:10px; text-anchor:start; alignment-baseline:hanging; fill:black;}</style>
<text class="capigraph-rendering-stats" x="0" y="0">%s</text>
<text class="capigraph-rendering-stats" x="20" y="0">%s</text>
</svg>`, errOpt.Error(), errUnopt.Error())
		}
	}

	return svg

}
