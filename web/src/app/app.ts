import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';

import { WikiHostComponent } from './wiki/wiki-host.component';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, WikiHostComponent],
  templateUrl: './app.html',
  styleUrl: './app.scss',
})
export class App {}
